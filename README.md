# Helm Operator
> This operator was built with [kubebuiler](https://book.kubebuilder.io/introduction.html)

This operator is a project to try move Helm Charts into a more declarative pattern, the understanding being that you declare your chart in a manifest then Kubernetes applies the resources around this chart (and attaches them to your chart), this gives the following upsides:

- Define RBAC roles to allow the operator to only perform certain actions/create certain types of resources
- Define Charts in a declarative way that can be easily version controlled (leaning towards a more GitOps Model)
- Use RBAC to give users access to create charts 

## Example Chart

```yaml
apiVersion: stable.helm.operator.io/v1
kind: Chart
metadata:
  name: nginx
spec:
  chart: nginx-ingress
  repo: stable
  version: 1.1.0
  nameSpaceSelector: "default"
  values:
  - name: controller.name
    value: "foo"
  - name: controller.autoscaling.enabled
    value: "true"
  - name: controller.replicaCount
    value: "4"
```

## Run Locally
To run this operator locally (It will use your kube config defined by $KUBECONFIG)

Install the CRDS 
```
make install
```

Run the operator
```
make run
```

## Run Example Chart
To run the example that installs the nginx ingress
```bash
kubectl apply -f example-chart.yaml
```

To view the chart
```bash
kubectl get chart nginx -o yaml
```
This should give you a nice yaml output, whats important to notice is the list of resources that have been created due to this chart
```yaml
apiVersion: stable.helm.operator.io/v1
kind: Chart
metadata:
    ...
spec:
  chart: nginx-ingress
  nameSpaceSelector: default
  repo: stable
  values:
  - name: controller.name
    value: foo
  - name: controller.autoscaling.enabled
    value: "true"
  - name: controller.replicaCount
    value: "4"
  version: 1.1.0
status:
  resource:
  - apiVersion: policy/v1beta1
    kind: PodDisruptionBudget
    name: nginx-nginx-ingress-foo
    namespace: default
  - apiVersion: v1
    kind: ConfigMap
    name: nginx-nginx-ingress-foo
    namespace: default
  - apiVersion: v1
    kind: ServiceAccount
    name: nginx-nginx-ingress
    namespace: default
  - apiVersion: rbac.authorization.k8s.io/v1beta1
    kind: ClusterRole
    name: nginx-nginx-ingress
    namespace: default
  - apiVersion: rbac.authorization.k8s.io/v1beta1
    kind: ClusterRoleBinding
    name: nginx-nginx-ingress
    namespace: default
  - apiVersion: rbac.authorization.k8s.io/v1beta1
    kind: Role
    name: nginx-nginx-ingress
    namespace: default
  - apiVersion: rbac.authorization.k8s.io/v1beta1
    kind: RoleBinding
    name: nginx-nginx-ingress
    namespace: default
  - apiVersion: v1
    kind: Service
    name: nginx-nginx-ingress-foo
    namespace: default
  - apiVersion: v1
    kind: Service
    name: nginx-nginx-ingress-default-backend
    namespace: default
  - apiVersion: extensions/v1beta1
    kind: Deployment
    name: nginx-nginx-ingress-foo
    namespace: default
  - apiVersion: extensions/v1beta1
    kind: Deployment
    name: nginx-nginx-ingress-default-backend
    namespace: default
  - apiVersion: autoscaling/v2beta1
    kind: HorizontalPodAutoscaler
    name: nginx-nginx-ingress-foo
    namespace: default
  status: Deployed
```
## ROADMAP:

- Add tests
- Add updating of resources
- Add namespace to chart manifests
- Allow for alternative repos (currently only supports stable)
- Allow for creation of namespace if it does not exist yet

