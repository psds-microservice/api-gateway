# Build

- **k8s/** — Dockerfile for production image (Kubernetes).
- **local/** — Dockerfile and Air config for local development with hot reload.

Build image for k8s from project root:
```bash
docker build -f build/k8s/Dockerfile -t api-gateway:latest .
```
