# k8s-gpu-exporter

## Command flags
 - `address` : Address to listen on for web interface and telemetry.
 - `kubeconfig` : Absolute path to the kubeconfig file, default get config from pod binding ServiceAccount.

## Docker Build
Tips :
By default, after the nvidia-docker container is started, there will be a symbolic link : `/usr/lib/x86_64-linux-gnu/libnvidia-ml.so.1` in the container by default, and destination of the symbolic link is a correct version `libnvidia-ml.so`. you can jump to step 4



If you don't have a correct version `libnvidia-ml.so` in container, you can solve it according to the following steps.


1. Find out the `libnvidia-ml.so` on you host which you want run this k8s-gpu-exporter.
    ```shell
    find /usr/ -name libnvidia-ml.so 
    ```
    
2. Copy the `libnvidia-ml.so` under `project-dir/lib` directory

3. Add line `COPY lib/libnvidia-ml.so /usr/lib/x86_64-linux-gnu/libnvidia-ml.so` in docker/dockerfile, for example:
    ```
    ...
    COPY --from=build-env /build/k8s-gpu-exporter /app/k8s-gpu-exporter
    COPY lib/libnvidia-ml.so /usr/lib/x86_64-linux-gnu/libnvidia-ml.so
    ... 
    ```

4. Run MakeFile
    ```shell
    VERSION={YOU_VERSION} make docker 
    ```

## Best Practices

If you already have [Arena](https://github.com/kubeflow/arena), use it to submit a training task.
```bash
# Preparation
# Label the GPU Node
$ kubectl lebel node {YOU_NODE}
# First
$ kubectl apply -f k8s-gpu-exporter.yaml

# Second
# """""""""""""""""""""""""""""""""""
# " Then expose this svc at {PORT}  "
# """""""""""""""""""""""""""""""""""

# Third
# Submit a deeplearn job
$ arena submit tf --name=style-transfer \
              --gpus=1 \
              --workers=1 \
              --workerImage=registry.cn-hangzhou.aliyuncs.com/tensorflow-samples/neural-style:gpu \
              --workingDir=/neural-style \
              --ps=1 \
              --psImage=registry.cn-hangzhou.aliyuncs.com/tensorflow-samples/style-transfer:ps   \
              "python neural_style.py --styles /neural-style/examples/1-style.jpg --iterations 1000"

# Fourth
$ curl {HOST_IP}:{PORT}/metrics
    
    ...Omit...
    # HELP nvidia_gpu_used_memory Graphics used memory 
    # TYPE nvidia_gpu_used_memory gauge
    nvidia_gpu_used_memory{gpu_node="dev-ms-7c22",gpu_pod_name="",minor_number="0",name="GeForce GTX 1660 SUPER",namepace_name="",uuid="GPU-a1460327-d919-1478-a68f-ef4cbb8515ac"} 3.0769152e+08
    nvidia_gpu_used_memory{gpu_node="dev-ms-7c22",gpu_pod_name="style-transfer-worker-0",minor_number="0",name="GeForce GTX 1660 SUPER",namepace_name="default",uuid="GPU-a1460327-d919-1478-a68f-ef4cbb8515ac"} 8.912896e+07
    ...Omit...

# Fifth
$ kubectl logs {YOUR_K8S_GPU_EXPORTER_POD}
    SystemGetDriverVersion: 450.36.06
    Not specify a config ,use default svc
    :9445
    We have 1 cards
    GPU-0   DeviceGetComputeRunningProcesses: 1
                    pid: 3598, usedMemory: 89128960 
                    node: dev-ms-7c22 pod: style-transfer-worker-0, pid: 3598 usedMemory: 89128960 
```



## Prometheus
Add Annotation `prometheus.io/scrape: 'true'` to k8s-gpu-exporter pod so that prometheus can automatically discover metrics services

And you can use PromQL query statement `nvidia_gpu_used_memory/nvidia_gpu_total_memory` to see gpu memory usage


![gpu_memory_usage](.github/image/gpu_memory_usage.png)

