# K8s Events Reporter 

Minimum implementation on how to gather events from a kubernetes cluster
and send the reports to a list of email recipients.

## Running the tests

```console
$ make test
```

## Deployment

To start minikube and deploy the events-reporter

```console
$ make start-deploy
```


## Configuration Example

```yaml
email_settings:
    smtp_server: mysmtp.server.com
    port: 25
    username: smtp_username
    password: smtp_password
reports:
  my_pod_report:
    email_recipients: ["test@test.com","test2@test.com"]
    description: Started pods
    kind: Pod
    reasons: ["Created","Started"]
    interval: 10h
  my_replicaset_report:
    email_recipients: ["test@test.com","test2@test.com"]
    description: ReplicaSets Created & Deleted
    kind: ReplicaSet
    reasons: ["SuccessfulCreate","SuccessfulDelete"]
    interval: 2h
```


## Built With

* [Golang](https://golang.org/) - The programming languange used.
* [Kubernetes](https://github.com/kubernetes/kubernetes) - Kubernetes libraries used.
* [logrus](https://github.com/sirupsen/logrus) - Log library used.
* [yaml](github.com/ghodss/yaml) - YAML library used.
* [gomail.v2](gopkg.in/gomail.v2) - mail client used.


## Authors

* **Nikolaos Moraitis** - *Initial work* - [droslean](https://github.com/droslean)

## License

This project is licensed under the GPL License - see the [LICENSE.md](LICENSE.md) file for details

