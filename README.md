# kubewatch

Kubernetes API event watcher.

##### Install

```
go get -u github.com/softonic/kubewatch
```

##### Shell completion

```
eval "$(kubewatch --completion-script-${0#-})"
```

##### Help

```
kubewatch --help
usage: kubewatch [<flags>]

Watches Kubernetes resources via its API.

Flags:
  -h, --help                 Show context-sensitive help (also try --help-long and --help-man).
      --kubeconfig           Absolute path to the kubeconfig file.
      --resource="services"  Set the resource type to be watched.
      --namespace=""         Set the namespace to be watched.
      --version              Show application version.
```

##### Examples:

Watch for `pods` events in all `namespaces`:
```
kubewatch --resource pods
```

Watch for `services` events in namespace `foo`:
```
kubewatch --namespace foo --resource services
```