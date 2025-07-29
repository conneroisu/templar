# Production Deployment Guide

This guide covers deploying Templar in production environments, including Docker, Kubernetes, cloud platforms, and monitoring setup.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Cloud Platform Deployment](#cloud-platform-deployment)
- [Monitoring and Observability](#monitoring-and-observability)
- [Security Considerations](#security-considerations)
- [Performance Tuning](#performance-tuning)
- [Troubleshooting](#troubleshooting)

## Prerequisites

Before deploying Templar in production, ensure you have:

- Go 1.24+ (for building from source)
- Docker and Docker Compose (for containerized deployments)
- Kubernetes cluster access (for K8s deployments)
- Cloud platform CLI tools (AWS CLI, gcloud, az CLI)
- SSL/TLS certificates for HTTPS
- Monitoring infrastructure (Prometheus, Grafana)

## Docker Deployment

### Basic Docker Deployment

Templar includes a production-ready Dockerfile for containerized deployments:

```dockerfile
# Build stage
FROM golang:1.24.4-alpine AS builder

# Install necessary packages
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Install templ for code generation (pinned version for security)
RUN go install github.com/a-h/templ/cmd/templ@v0.3.819

# Copy source code
COPY . .

# Generate Go code from templates
RUN go generate ./...

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags '-extldflags "-static" -s -w' \
    -o templar .

# Final stage
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo/
COPY --from=builder /app/templar /templar

# Create a non-root user
USER 1000:1000

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/templar", "health"] || exit 1

# Set entrypoint
ENTRYPOINT ["/templar"]
CMD ["serve"]
```

### Docker Compose Production Setup

Use the following `docker-compose.prod.yml` for a complete production stack:

```yaml
version: '3.8'

services:
  templar:
    build: 
      context: .
      dockerfile: Dockerfile
      target: production
    ports:
      - "8080:8080"
    volumes:
      - ./components:/app/components:ro
      - ./static:/app/static:ro
      - templar-cache:/app/.templar/cache
      - ./config/production.yml:/app/.templar.yml:ro
    environment:
      - TEMPLAR_ENV=production
      - TEMPLAR_LOG_LEVEL=info
      - TEMPLAR_CACHE_SIZE=10000
      - TEMPLAR_SERVER_HOST=0.0.0.0
      - TEMPLAR_MONITORING_ENABLED=true
      - TEMPLAR_MONITORING_HTTP_PORT=8081
    networks:
      - templar-network
    healthcheck:
      test: ["CMD", "/templar", "health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    restart: unless-stopped
    deploy:
      replicas: 2
      resources:
        limits:
          memory: 512M
          cpus: '0.5'
        reservations:
          memory: 256M
          cpus: '0.25'

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./config/nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./config/nginx/ssl:/etc/nginx/ssl:ro
      - ./static:/var/www/static:ro
    depends_on:
      templar:
        condition: service_healthy
    networks:
      - templar-network
    restart: unless-stopped

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./config/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--web.enable-lifecycle'
      - '--storage.tsdb.retention.time=30d'
    networks:
      - templar-network
    restart: unless-stopped

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - grafana-data:/var/lib/grafana
      - ./config/grafana/dashboards:/etc/grafana/provisioning/dashboards:ro
      - ./config/grafana/datasources:/etc/grafana/provisioning/datasources:ro
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_ADMIN_PASSWORD}
      - GF_SERVER_ROOT_URL=https://monitoring.example.com
      - GF_SECURITY_SECRET_KEY=${GRAFANA_SECRET_KEY}
    depends_on:
      - prometheus
    networks:
      - templar-network
    restart: unless-stopped

  redis:
    image: redis:alpine
    volumes:
      - redis-data:/data
    networks:
      - templar-network
    restart: unless-stopped
    command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru

networks:
  templar-network:
    driver: bridge

volumes:
  templar-cache:
  prometheus-data:
  grafana-data:
  redis-data:
```

### Production Configuration

Create `config/production.yml`:

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  open: false
  environment: "production"
  middleware:
    - cors
    - logger
    - security
    - compression
  allowed_origins:
    - "https://yourdomain.com"
    - "https://www.yourdomain.com"
  auth:
    enabled: true
    mode: "token"
    token: "${TEMPLAR_AUTH_TOKEN}"
    require_auth: true
    localhost_bypass: false

build:
  command: "templ generate"
  cache_dir: "/app/.templar/cache"

components:
  scan_paths: ["/app/components"]
  exclude_patterns: ["*_test.templ", "*.bak"]

development:
  hot_reload: false
  css_injection: false
  error_overlay: false

production:
  output_dir: "/app/dist"
  static_dir: "/app/static"
  
  minification:
    css: true
    javascript: true
    html: true
    remove_comments: true
    strip_debug: true
  
  compression:
    enabled: true
    algorithms: ["gzip", "brotli"]
    level: 6
    extensions: [".html", ".css", ".js", ".json", ".xml", ".svg"]
  
  asset_optimization:
    critical_css: true
    tree_shaking: true
    images:
      enabled: true
      quality: 85
      progressive: true
      formats: ["webp", "avif"]
  
  security:
    hsts: true
    x_frame_options: "DENY"
    x_content_type_options: true
    csp:
      enabled: true
      directives:
        default-src: "'self'"
        script-src: "'self' 'unsafe-inline'"
        style-src: "'self' 'unsafe-inline'"
        img-src: "'self' data: https:"
  
  deployment:
    target: "docker"
    environment: "production"
    base_url: "https://yourdomain.com"

monitoring:
  enabled: true
  log_level: "info"
  log_format: "json"
  metrics_path: "/app/logs/metrics.json"
  http_port: 8081

timeouts:
  build: "5m"
  external: "3m"
  network: "30s"
  http: "30s"
  websocket: "60s"
  startup: "30s"
  shutdown: "30s"
```

## Kubernetes Deployment

### Namespace and Service Account

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: templar
  labels:
    name: templar
    app.kubernetes.io/name: templar
    app.kubernetes.io/component: application

---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: templar
  namespace: templar
  labels:
    app.kubernetes.io/name: templar
    app.kubernetes.io/component: service-account
```

### ConfigMap and Secrets

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: templar-config
  namespace: templar
data:
  templar.yml: |
    server:
      port: 8080
      host: "0.0.0.0"
      environment: "production"
    monitoring:
      enabled: true
      log_level: "info"
      http_port: 8081
    production:
      minification:
        css: true
        javascript: true
        html: true
      compression:
        enabled: true
        algorithms: ["gzip", "brotli"]
      security:
        hsts: true
        x_frame_options: "DENY"

---
apiVersion: v1
kind: Secret
metadata:
  name: templar-secrets
  namespace: templar
type: Opaque
data:
  auth-token: <base64-encoded-token>
  grafana-password: <base64-encoded-password>
```

### Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: templar
  namespace: templar
  labels:
    app.kubernetes.io/name: templar
    app.kubernetes.io/component: application
    app.kubernetes.io/version: "1.14.9"
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: templar
  template:
    metadata:
      labels:
        app.kubernetes.io/name: templar
        app.kubernetes.io/component: application
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8081"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: templar
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        runAsGroup: 1000
        fsGroup: 1000
      containers:
      - name: templar
        image: templar:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8080
          name: http
          protocol: TCP
        - containerPort: 8081
          name: metrics
          protocol: TCP
        env:
        - name: TEMPLAR_ENV
          value: "production"
        - name: TEMPLAR_AUTH_TOKEN
          valueFrom:
            secretKeyRef:
              name: templar-secrets
              key: auth-token
        - name: TEMPLAR_LOG_LEVEL
          value: "info"
        - name: TEMPLAR_MONITORING_ENABLED
          value: "true"
        volumeMounts:
        - name: config
          mountPath: /app/.templar.yml
          subPath: templar.yml
          readOnly: true
        - name: cache
          mountPath: /app/.templar/cache
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 2
      volumes:
      - name: config
        configMap:
          name: templar-config
      - name: cache
        emptyDir:
          sizeLimit: 1Gi
```

### Service and Ingress

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: templar
  namespace: templar
  labels:
    app.kubernetes.io/name: templar
    app.kubernetes.io/component: service
spec:
  type: ClusterIP
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
    name: http
  - port: 8081
    targetPort: 8081
    protocol: TCP
    name: metrics
  selector:
    app.kubernetes.io/name: templar

---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: templar
  namespace: templar
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"
    nginx.ingress.kubernetes.io/rate-limit: "100"
spec:
  tls:
  - hosts:
    - templar.yourdomain.com
    secretName: templar-tls
  rules:
  - host: templar.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: templar
            port:
              number: 80
```

### Horizontal Pod Autoscaler

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: templar
  namespace: templar
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: templar
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
```

## Cloud Platform Deployment

### AWS ECS Deployment

#### Task Definition

```json
{
  "family": "templar",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "executionRoleArn": "arn:aws:iam::ACCOUNT:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::ACCOUNT:role/templarTaskRole",
  "containerDefinitions": [
    {
      "name": "templar",
      "image": "your-registry/templar:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "TEMPLAR_ENV",
          "value": "production"
        },
        {
          "name": "TEMPLAR_LOG_LEVEL",
          "value": "info"
        }
      ],
      "secrets": [
        {
          "name": "TEMPLAR_AUTH_TOKEN",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:templar/auth-token"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/templar",
          "awslogs-region": "us-west-2",
          "awslogs-stream-prefix": "ecs"
        }
      },
      "healthCheck": {
        "command": ["CMD-SHELL", "/templar health || exit 1"],
        "interval": 30,
        "timeout": 5,
        "retries": 3,
        "startPeriod": 60
      }
    }
  ]
}
```

#### CloudFormation Template

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Description: 'Templar ECS Deployment'

Parameters:
  VpcId:
    Type: AWS::EC2::VPC::Id
    Description: VPC for ECS cluster
  SubnetIds:
    Type: List<AWS::EC2::Subnet::Id>
    Description: Subnets for ECS tasks
  ImageURI:
    Type: String
    Description: Docker image URI

Resources:
  ECSCluster:
    Type: AWS::ECS::Cluster
    Properties:
      ClusterName: templar-cluster
      CapacityProviders:
        - FARGATE
        - FARGATE_SPOT
      DefaultCapacityProviderStrategy:
        - CapacityProvider: FARGATE
          Weight: 1
        - CapacityProvider: FARGATE_SPOT
          Weight: 4

  ALB:
    Type: AWS::ElasticLoadBalancingV2::LoadBalancer
    Properties:
      Name: templar-alb
      Scheme: internet-facing
      Type: application
      Subnets: !Ref SubnetIds
      SecurityGroups:
        - !Ref ALBSecurityGroup

  ECSService:
    Type: AWS::ECS::Service
    Properties:
      ServiceName: templar-service
      Cluster: !Ref ECSCluster
      TaskDefinition: !Ref TaskDefinition
      DesiredCount: 3
      LaunchType: FARGATE
      NetworkConfiguration:
        AwsvpcConfiguration:
          Subnets: !Ref SubnetIds
          SecurityGroups:
            - !Ref TaskSecurityGroup
          AssignPublicIp: ENABLED
      LoadBalancers:
        - ContainerName: templar
          ContainerPort: 8080
          TargetGroupArn: !Ref TargetGroup
```

### Google Cloud Run Deployment

```yaml
# service.yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: templar
  namespace: default
  annotations:
    run.googleapis.com/ingress: all
    run.googleapis.com/execution-environment: gen2
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/maxScale: "10"
        autoscaling.knative.dev/minScale: "1"
        run.googleapis.com/cpu-throttling: "false"
        run.googleapis.com/execution-environment: gen2
        run.googleapis.com/memory: "1Gi"
        run.googleapis.com/cpu: "1000m"
    spec:
      containerConcurrency: 80
      timeoutSeconds: 300
      containers:
      - image: gcr.io/PROJECT_ID/templar:latest
        ports:
        - containerPort: 8080
        env:
        - name: TEMPLAR_ENV
          value: "production"
        - name: TEMPLAR_LOG_LEVEL
          value: "info"
        - name: TEMPLAR_AUTH_TOKEN
          valueFrom:
            secretKeyRef:
              key: token
              name: templar-secrets
        resources:
          limits:
            cpu: 1000m
            memory: 1Gi
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
```

### Azure Container Instances

```yaml
# deployment.yaml
apiVersion: '2019-12-01'
location: eastus
name: templar-container-group
properties:
  containers:
  - name: templar
    properties:
      image: yourregistry.azurecr.io/templar:latest
      resources:
        requests:
          cpu: 1
          memoryInGB: 1
      ports:
      - port: 8080
        protocol: TCP
      environmentVariables:
      - name: TEMPLAR_ENV
        value: production
      - name: TEMPLAR_LOG_LEVEL
        value: info
      - name: TEMPLAR_AUTH_TOKEN
        secureValue: your-secret-token
  osType: Linux
  restartPolicy: Always
  ipAddress:
    type: Public
    ports:
    - protocol: TCP
      port: 8080
    dnsNameLabel: templar-app
type: Microsoft.ContainerInstance/containerGroups
```

## Monitoring and Observability

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "templar_rules.yml"

scrape_configs:
  - job_name: 'templar'
    static_configs:
      - targets: ['templar:8081']
    scrape_interval: 10s
    metrics_path: /metrics
    
  - job_name: 'templar-health'
    static_configs:
      - targets: ['templar:8080']
    scrape_interval: 30s
    metrics_path: /health
    
alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "id": null,
    "title": "Templar Performance Dashboard",
    "tags": ["templar", "performance"],
    "timezone": "browser",
    "panels": [
      {
        "id": 1,
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(templar_http_requests_total[5m])",
            "legendFormat": "Requests/sec"
          }
        ]
      },
      {
        "id": 2,
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(templar_http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          }
        ]
      },
      {
        "id": 3,
        "title": "Build Performance",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(templar_build_duration_seconds[5m])",
            "legendFormat": "Build time"
          }
        ]
      }
    ]
  }
}
```

### OpenTelemetry Configuration

```yaml
# otel-collector.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
  resourcedetection:
    detectors: [env, system]

exporters:
  prometheus:
    endpoint: "0.0.0.0:8889"
  jaeger:
    endpoint: jaeger-collector:14250
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [jaeger]
    metrics:
      receivers: [otlp]
      processors: [batch, resourcedetection]
      exporters: [prometheus]
```

## Security Considerations

### Network Security

1. **Use HTTPS everywhere**:
   ```yaml
   server:
     tls:
       enabled: true
       cert_file: /etc/ssl/certs/templar.crt
       key_file: /etc/ssl/private/templar.key
   ```

2. **Configure strict CORS**:
   ```yaml
   server:
     allowed_origins:
       - "https://yourdomain.com"
       - "https://www.yourdomain.com"
   ```

3. **Enable security headers**:
   ```yaml
   production:
     security:
       hsts: true
       x_frame_options: "DENY"
       x_content_type_options: true
       csp:
         enabled: true
         directives:
           default-src: "'self'"
           script-src: "'self' 'unsafe-inline'"
           style-src: "'self' 'unsafe-inline'"
   ```

### Authentication and Authorization

1. **Strong authentication**:
   ```yaml
   server:
     auth:
       enabled: true
       mode: "token"
       token: "${TEMPLAR_AUTH_TOKEN}" # Use environment variable
       require_auth: true
       localhost_bypass: false
   ```

2. **IP allowlisting**:
   ```yaml
   server:
     auth:
       allowed_ips:
         - "10.0.0.0/8"
         - "192.168.0.0/16"
   ```

### Container Security

1. **Run as non-root user**:
   ```dockerfile
   USER 1000:1000
   ```

2. **Use distroless images**:
   ```dockerfile
   FROM gcr.io/distroless/static:nonroot
   ```

3. **Scan for vulnerabilities**:
   ```bash
   docker scan templar:latest
   trivy image templar:latest
   ```

## Performance Tuning

### Application Performance

1. **Cache configuration**:
   ```yaml
   build:
     cache_dir: "/app/.templar/cache"
   
   production:
     performance:
       budget_limits:
         bundle_size: 500000  # 500KB
         image_size: 1000000  # 1MB
   ```

2. **Worker pool optimization**:
   ```yaml
   build:
     workers: 4  # Match CPU cores
     timeout: "5m"
   ```

3. **Memory limits**:
   ```yaml
   # Kubernetes
   resources:
     limits:
       memory: "512Mi"
       cpu: "500m"
     requests:
       memory: "256Mi"
       cpu: "250m"
   ```

### Database Performance

For applications using external databases:

```yaml
database:
  max_connections: 20
  idle_timeout: "5m"
  connection_timeout: "30s"
```

### CDN Configuration

```yaml
production:
  cdn:
    enabled: true
    provider: "cloudflare"
    base_path: "https://cdn.yourdomain.com"
    cache_ttl: 86400  # 24 hours
    invalidation: true
```

## Troubleshooting

### Common Issues

1. **Build failures**:
   ```bash
   # Check build logs
   docker logs templar-container --tail 100
   
   # Verify templ installation
   docker exec templar-container /bin/sh -c "which templ"
   ```

2. **Memory issues**:
   ```bash
   # Monitor memory usage
   kubectl top pods -n templar
   
   # Check for memory leaks
   docker stats templar-container
   ```

3. **Network connectivity**:
   ```bash
   # Test health endpoint
   curl -f http://localhost:8080/health
   
   # Check DNS resolution
   nslookup templar.yourdomain.com
   ```

### Debugging

1. **Enable debug logging**:
   ```yaml
   monitoring:
     log_level: "debug"
   ```

2. **Health check endpoints**:
   - `/health` - Basic health check
   - `/ready` - Readiness probe
   - `/metrics` - Prometheus metrics

3. **Performance profiling**:
   ```bash
   # Enable pprof in production
   TEMPLAR_PPROF_ENABLED=true templar serve
   
   # Access profiling data
   go tool pprof http://localhost:8080/debug/pprof/profile
   ```

### Log Analysis

1. **Structured logging**:
   ```yaml
   monitoring:
     log_format: "json"
     log_level: "info"
   ```

2. **Log aggregation with ELK Stack**:
   ```yaml
   logging:
     driver: "fluentd"
     options:
       fluentd-address: "fluentd:24224"
       tag: "templar.{{.Name}}"
   ```

### Support

For additional support:

1. Check the [troubleshooting guide](./TROUBLESHOOTING.md)
2. Review [performance benchmarks](./docs/performance/README.md)
3. Open an issue on [GitHub](https://github.com/conneroisu/templar/issues)
4. Join our [Discord community](https://discord.gg/templar)

---

This deployment guide provides comprehensive coverage for production deployments. Adjust configurations based on your specific requirements and infrastructure setup.