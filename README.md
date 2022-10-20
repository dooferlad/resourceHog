# Resource Hog

I needed something to use resources to test Kubernetes scaling, so here it is.
This is a hack day project that has been hand tested, but doesn't have unit or
functional tests.

## Installation
`go get github.com/dooferlad/resourceHog`

## Run
Run the `./resourceHog` binary where you want resources to be used.

To control the hog, call `<server>:6776/` with one or more of the following query parameters

 * `time=<duration>`
 * `cpu=<number of CPUs to hog>`
 * `ram=<size>`
 * `disk_write=<size>`
 * `disk_read=<size>`
 * `response_size=<size>`

All sizes are parsed as standard computer units, so you can use MB, GiB etc.
Times are parsed using time.ParseDuration, so you can specify time with units
from nanoseconds up to hours.

The time parameter is needed for CPU and RAM hogs to tell the hog how much of those
resources to use, and for how long.

A note on RAM hogs - the code has been tested and RAM is cleaned up when we want
it to under test, but since Go doesn't allow direct memory management a change in
the runtime may change this. I wanted to keep this simple, so I haven't used an
alternative GC or called out to C code to malloc and free.

## Using in K8s

You probably want to deploy something like this
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: resourcehog
  labels:
    app: resourcehog
spec:
  replicas: 1
  selector:
    matchLabels:
      app: resourcehog
  template:
    metadata:
      labels:
        app: resourcehog
    spec:
      containers:
        - name: resourcehog
          image: dooferlad/resourcehog:1.0
          ports:
            - containerPort: 6776
          resources:
            limits:
              memory: "512Mi"
              cpu: "1"
            requests:
              cpu: "800m"
---
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: resourcehog
spec:
  scaleTargetRef:
    name: resourcehog
  pollingInterval:  1
  cooldownPeriod:   2
  minReplicaCount:  1
  maxReplicaCount:  5
  triggers:
  - type: cpu
    metricType: Utilization # Allowed types are 'Utilization' or 'AverageValue'
    metadata:
      value: "60"
---
apiVersion: v1
kind: Service
metadata:
  name: resourcehog
  labels:
    app: resourcehog
spec:
  selector:
    app: resourcehog
  ports:
    - protocol: TCP
      port: 6776
```

Then jump onto the cluster and use some resources
```
$ kubectl describe svc resourcehog
Name:              resourcehog
Namespace:         default
Labels:            app=resourcehog
Annotations:       <none>
Selector:          app=resourcehog
Type:              ClusterIP
IP Family Policy:  SingleStack
IP Families:       IPv4
IP:                <service IP address> <-- Note this
IPs:               
Port:              <unset>  6776/TCP
TargetPort:        6776/TCP
Endpoints:         10.0.221.113:6776
Session Affinity:  None
Events:            <none>

$ kubectl run curl --image=radial/busyboxplus:curl -i --tty
If you don't see a command prompt, try pressing enter.
[ root@curl:/ ]$ curl "<service IP address>:6776/?cpu=1&time=600s" &
```

At this point you should see another resourcehog pod starting up. You can keep running the above curl command
in the busybox container to increase the resources being used. Kubernetes will spread the requests across active
replicas. If you have the cluster autoscaler set up to deploy more worker machines, you should see this happening
as they fill up.