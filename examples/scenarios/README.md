# KubeDepGuard 實驗情境

先完成 kind 部署，並確認目前 context 是目標 kind cluster：

```sh
kubectl config current-context
kubectl get pods -n kube-dep-guard-system
```

所有案例使用目前 namespace；若要隔離實驗，可先建立一個 namespace，再在各指令加上 `-n <namespace>`。

| 情境 | 預期結果 | 驗證面向 |
| --- | --- | --- |
| Enforce Pod 引用不存在 ConfigMap | `kubectl apply` 被拒絕 | Direct reference admission |
| Enforce Deployment template 引用不存在 ConfigMap | `kubectl apply` 被拒絕 | PodSpec-based direct admission |
| Enforce Pod 引用不存在 PVC | `kubectl apply` 被拒絕 | Direct reference admission |
| Warn Pod 使用尚未 Bound 的 PVC | 建立成功，Monitor 發 Warning | Direct reference + state |
| Warn Pod 引用不存在 ConfigMap | 建立成功，Monitor 發 Warning | Runtime direct reference |
| Enforce Service selector 為空 | `kubectl apply` 被拒絕 | Conditional admission |
| Enforce ConfigMap deletion | `kubectl delete configmap` 被拒絕 | Direct dependency delete protection |
| Enforce final Pod deletion | `kubectl delete pod` 被拒絕 | Conditional dependency delete protection |
| Service 無 Ready endpoint | 建立成功，Monitor 發 Warning | Runtime availability |

## 1. Enforce Pod 引用不存在的 ConfigMap

```sh
kubectl apply -f examples/failures/enforce-pod-missing-configmap.yaml
```

預期：Admission Webhook 拒絕請求，訊息包含 `references ConfigMap`。

## 2. Enforce Deployment template 引用不存在的 ConfigMap

```sh
kubectl apply -f examples/failures/enforce-deployment-missing-configmap.yaml
```

預期：Deployment 建立前就被 Admission Webhook 拒絕；不必等 ReplicaSet 或 Pod 建立。

## 3. Warn Pod 引用不存在的 ConfigMap

```sh
kubectl apply -f examples/failures/warn-pod-missing-configmap.yaml
kubectl get events --field-selector reason=MissingConfigMap --sort-by=.lastTimestamp
```

預期：Pod 建立成功；Monitor 建立 `MissingConfigMap` Warning Event。

## 4. Enforce Pod 引用不存在的 PVC

```sh
kubectl apply -f examples/failures/enforce-pod-missing-pvc.yaml
```

預期：Admission Webhook 拒絕請求，訊息包含 `PersistentVolumeClaim`。

## 5. Warn Pod 使用尚未 Bound 的 PVC

```sh
kubectl apply -f examples/failures/warn-pod-pvc-pending.yaml
kubectl get events --field-selector reason=PersistentVolumeClaimNotBound --sort-by=.lastTimestamp
```

預期：Pod 建立成功；PVC 維持 Pending，Monitor 建立 `PersistentVolumeClaimNotBound` Warning Event。

## 6. Enforce Service selector 沒有匹配 Pod

```sh
kubectl apply -f examples/failures/enforce-service-empty-selector.yaml
```

預期：Admission Webhook 拒絕請求，訊息包含 `selector matches no Pods`。

## 7. 阻擋刪除仍被 enforce Pod 使用的 ConfigMap

```sh
kubectl apply -f examples/failures/enforce-configmap-delete-setup.yaml
kubectl delete configmap protected-config
```

預期：第二個命令被拒絕，因為 `enforce-configmap-consumer` 仍引用 `protected-config`。

清理時，先刪除 Pod，再刪除 ConfigMap：

```sh
kubectl delete pod enforce-configmap-consumer
kubectl delete configmap protected-config
```

## 8. 阻擋刪除 enforce Service 的最後一個 Pod

```sh
kubectl apply -f examples/failures/enforce-last-pod-delete-setup.yaml
kubectl delete pod enforce-service-target
```

預期：第二個命令被拒絕，因為 Service 將失去最後一個符合 selector 的 Pod。

清理時先刪除 Service：

```sh
kubectl delete service enforce-service-target
kubectl delete pod enforce-service-target
```

## 9. Service 沒有 Ready endpoint

```sh
kubectl apply -f examples/failures/warn-service-no-ready-endpoint.yaml
kubectl get events --field-selector reason=NoReadyEndpoint --sort-by=.lastTimestamp
```

預期：Pod 因固定失敗的 readiness probe 維持 NotReady；Service 仍會建立，但 Monitor 發出 `NoReadyEndpoint` Warning Event。

清理：

```sh
kubectl delete -f examples/failures/warn-service-no-ready-endpoint.yaml
```
