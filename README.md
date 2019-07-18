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

## ROADMAP:

- Add tests
- Add updating of resources
- Add namespace to chart manifests
- Allow for alternative repos (currently only supposrts stable)

