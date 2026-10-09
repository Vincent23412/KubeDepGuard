# KubeDepGuard 實驗情境

先完成 kind 部署，並確認目前 context 是目標 kind cluster：

```sh
kubectl config current-context
kubectl get pods -n kube-dep-guard-system
```

所有案例使用目前 namespace；若要隔離實驗，可先建立一個 namespace，再在各指令加上 `-n <namespace>`。

| 情境 | 預期結果 | 驗證面向 |
| --- | --- | --- |
| Enforce Deployment template 引用不存在 ConfigMap | `kubectl apply` 被拒絕 | PodSpec-based direct admission |
| Enforce Pod 引用不存在的 Secret key | `kubectl apply` 被拒絕 | Direct reference key validation |
| Enforce Deployment 引用不存在 PVC | `kubectl apply` 被拒絕 | Direct reference admission |
| Enforce Service selector 為空 | `kubectl apply` 被拒絕 | Conditional admission |
| Enforce Ingress 引用不存在的 Service | `kubectl apply` 被拒絕 | Direct reference admission |
| Enforce Ingress 引用不存在的 Service port | `kubectl apply` 被拒絕 | Direct reference validation |
| Enforce ConfigMap deletion | `kubectl delete configmap` 被拒絕 | Direct dependency delete protection |
| Enforce final Pod deletion | `kubectl delete pod` 被拒絕 | Conditional dependency delete protection |
| Enforce PVC deletion | `kubectl delete pvc` 被拒絕 | Direct dependency delete protection |

## 1. Enforce Deployment template 引用不存在的 ConfigMap

```sh
kubectl apply -f examples/failures/enforce-deployment-missing-configmap.yaml
```

預期：Deployment 建立前就被 Admission Webhook 拒絕；不必等 ReplicaSet 或 Pod 建立。

## 2. Enforce Deployment template 引用不存在的 PVC

```sh
kubectl apply -f examples/failures/enforce-deployment-missing-pvc.yaml
```

預期：Admission Webhook 拒絕請求，訊息包含 `PersistentVolumeClaim`。

## 3. Enforce Pod 引用不存在的 Secret key

```sh
kubectl apply -f examples/failures/enforce-pod-missing-secret-key.yaml
```

預期：Secret 本身存在，但被引用的 key 不存在，因此 Admission Webhook 拒絕 Pod。

## 4. Enforce Service selector 沒有匹配 Pod

```sh
kubectl apply -f examples/failures/enforce-service-empty-selector.yaml
```

預期：Admission Webhook 拒絕請求，訊息包含 `selector matches no Pods`。

## 5. Enforce Ingress 引用不存在的 Service 或 port

```sh
kubectl apply -f examples/failures/enforce-ingress-missing-service.yaml
kubectl apply -f examples/failures/enforce-ingress-missing-service-port.yaml
```

預期：Admission Webhook 分別拒絕不存在的 backend Service，以及 Service 中不存在的 backend port。

## 6. 阻擋刪除仍被 enforce Deployment 使用的 ConfigMap

```sh
kubectl apply -f examples/failures/enforce-configmap-delete-setup.yaml
kubectl delete configmap protected-config
```

預期：第二個命令被拒絕，因為 `enforce-configmap-consumer` 仍引用 `protected-config`。

清理時，先刪除 Deployment，再刪除 ConfigMap：

```sh
kubectl delete deployment enforce-configmap-consumer
kubectl delete configmap protected-config
```

## 7. 阻擋刪除 enforce Service 的最後一個 Pod

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

## 8. 阻擋刪除仍被 enforce Deployment 使用的 PVC

```sh
kubectl apply -f examples/failures/enforce-pvc-delete-setup.yaml
kubectl delete persistentvolumeclaim protected-data
```

預期：第二個命令被拒絕，因為 `enforce-pvc-consumer` 仍引用 `protected-data`。
