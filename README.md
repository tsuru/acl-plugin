# acl-plugin
tsuru CLI plugin to interact with ACL-API
```mermaid
graph TD;
    developer[Developer];
    tsuru-client[Tsuru Client + ACL Plugin];
    aclapi[ACL-API];
    mongodb[(MongoDB)];
    acl-operator[acl-operator];
    network-policies[Kubernetes Network Policies]

    developer -- Manage ACL Rules --> tsuru-client;
    tsuru-client -- Calls via API --> tsuru
    tsuru -- service contract --> aclapi;
    aclapi --> mongodb;
    acl-operator -- Pull Rules ----> aclapi

    click tsuru-client "https://www.github.com/tsuru/tsuru-client" "Access github project"
    click tsuru "https://www.github.com/tsuru/tsuru" "Access github project"
    click aclapi "https://www.github.com/tsuru/acl-api" "Access github project"

    click acl-operator "https://www.github.com/tsuru/acl-operator" "Access github project"
    click network-policies "https://kubernetes.io/docs/concepts/services-networking/network-policies/" "Read more about kubernetes network policies"

    subgraph "cluster(s) [1..N]"
      acl-operator -- Manage --> network-policies
    end
```

## Install

```
tsuru plugin install acl https://github.com/tsuru/acl-plugin/releases/latest/download/manifest.json
```
