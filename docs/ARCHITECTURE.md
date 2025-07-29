# Templar Architecture Documentation

This document provides comprehensive architecture documentation for Templar, including high-level system architecture, component relationships, data flows, and key workflows with professional diagrams.

## Table of Contents

- [System Overview](#system-overview)
- [High-Level Architecture](#high-level-architecture)
- [Package Architecture](#package-architecture)
- [Core Component Interactions](#core-component-interactions)
- [Key Workflows](#key-workflows)
- [Data Flow Diagrams](#data-flow-diagrams)
- [Security Architecture](#security-architecture)
- [Performance Architecture](#performance-architecture)
- [Deployment Architecture](#deployment-architecture)

## System Overview

Templar is a rapid prototyping CLI tool for Go templ components that provides browser preview functionality, hot reload capability, and streamlined development workflows. The system is designed with a modular, event-driven architecture that prioritizes security, performance, and developer experience.

### Core Principles

- **Safety**: Defense-in-depth security with input validation, command injection prevention, and race condition protection
- **Performance**: Optimized for 30M+ operations/second with concurrent processing, object pooling, and lock-free operations
- **Developer Experience**: Clean interfaces, comprehensive testing, and zero technical debt policy

## High-Level Architecture

```mermaid
graph TB
    subgraph "CLI Layer"
        CLI[CLI Commands]
        FLAGS[Flag Parsing]
        ARGS[Argument Validation]
    end

    subgraph "Service Layer"
        INIT[Initialization Service]
        BUILD[Build Service]
        SERVE[Server Service]
    end

    subgraph "Core Engine"
        REG[Component Registry]
        SCAN[Component Scanner]
        WATCH[File Watcher]
        PIPE[Build Pipeline]
    end

    subgraph "Web Layer"
        SERVER[HTTP Server]
        WS[WebSocket Manager]
        UI[Web Interface]
        PLAYGROUND[Component Playground]
    end

    subgraph "Infrastructure"
        CONFIG[Configuration System]
        LOG[Logging System]
        ERR[Error Handling]
        MON[Monitoring System]
    end

    subgraph "External Systems"
        FS[File System]
        TEMPL[Templ Compiler]
        BROWSER[Web Browser]
    end

    CLI --> ARGS
    FLAGS --> CONFIG
    ARGS --> INIT
    ARGS --> BUILD
    ARGS --> SERVE

    INIT --> REG
    BUILD --> PIPE
    SERVE --> SERVER

    SCAN --> REG
    WATCH --> SCAN
    PIPE --> TEMPL
    REG --> SERVER
    
    SERVER --> WS
    SERVER --> UI
    UI --> PLAYGROUND
    WS --> BROWSER
    
    REG --> MON
    PIPE --> ERR
    SERVER --> LOG
    
    SCAN --> FS
    WATCH --> FS

    classDef cliLayer fill:#e1f5fe
    classDef serviceLayer fill:#f3e5f5
    classDef coreEngine fill:#e8f5e8
    classDef webLayer fill:#fff3e0
    classDef infrastructure fill:#fce4ec
    classDef external fill:#f5f5f5

    class CLI,FLAGS,ARGS cliLayer
    class INIT,BUILD,SERVE serviceLayer
    class REG,SCAN,WATCH,PIPE coreEngine
    class SERVER,WS,UI,PLAYGROUND webLayer
    class CONFIG,LOG,ERR,MON infrastructure
    class FS,TEMPL,BROWSER external
```

## Package Architecture

```mermaid
graph TB
    subgraph "cmd/"
        ROOT[root.go]
        INIT_CMD[init.go]
        SERVE_CMD[serve.go]
        BUILD_CMD[build.go]
        LIST_CMD[list.go]
        PREVIEW_CMD[preview.go]
    end

    subgraph "internal/"
        subgraph "Core Components"
            REGISTRY[registry/]
            SCANNER[scanner/]
            WATCHER[watcher/]
            BUILD_PKG[build/]
        end

        subgraph "Web Components"
            SERVER_PKG[server/]
            WEBSOCKET[websocket/]
            RENDERER[renderer/]
        end

        subgraph "Infrastructure"
            CONFIG_PKG[config/]
            ERRORS[errors/]
            LOGGING[logging/]
            MONITOR[monitoring/]
        end

        subgraph "Support"
            INTERFACES[interfaces/]
            TYPES[types/]
            TESTUTILS[testutils/]
            VALIDATION[validation/]
        end
    end

    ROOT --> INIT_CMD
    ROOT --> SERVE_CMD
    ROOT --> BUILD_CMD
    ROOT --> LIST_CMD
    ROOT --> PREVIEW_CMD

    INIT_CMD --> CONFIG_PKG
    INIT_CMD --> REGISTRY
    
    SERVE_CMD --> SERVER_PKG
    SERVE_CMD --> WATCHER
    
    BUILD_CMD --> BUILD_PKG
    BUILD_CMD --> SCANNER
    
    LIST_CMD --> REGISTRY
    PREVIEW_CMD --> RENDERER

    REGISTRY --> TYPES
    SCANNER --> REGISTRY
    WATCHER --> SCANNER
    BUILD_PKG --> ERRORS
    
    SERVER_PKG --> WEBSOCKET
    SERVER_PKG --> RENDERER
    WEBSOCKET --> MONITOR
    
    CONFIG_PKG --> VALIDATION
    ERRORS --> LOGGING
    MONITOR --> LOGGING

    classDef cmdPkg fill:#e1f5fe
    classDef corePkg fill:#e8f5e8
    classDef webPkg fill:#fff3e0
    classDef infraPkg fill:#fce4ec
    classDef supportPkg fill:#f5f5f5

    class ROOT,INIT_CMD,SERVE_CMD,BUILD_CMD,LIST_CMD,PREVIEW_CMD cmdPkg
    class REGISTRY,SCANNER,WATCHER,BUILD_PKG corePkg
    class SERVER_PKG,WEBSOCKET,RENDERER webPkg
    class CONFIG_PKG,ERRORS,LOGGING,MONITOR infraPkg
    class INTERFACES,TYPES,TESTUTILS,VALIDATION supportPkg
```

## Core Component Interactions

```mermaid
graph LR
    subgraph "Component Discovery Flow"
        FS[File System] --> SCANNER[Component Scanner]
        SCANNER --> REG[Component Registry]
        REG --> CACHE[Registry Cache]
    end

    subgraph "Development Server Flow"
        REG --> SERVER[HTTP Server]
        SERVER --> WS[WebSocket Manager]
        WS --> BROWSER[Browser]
        
        WATCHER[File Watcher] --> SCANNER
        SCANNER --> BUILD[Build Pipeline]
        BUILD --> WS
    end

    subgraph "Build Process Flow"
        BUILD --> TEMPL[Templ Compiler]
        TEMPL --> OUTPUT[Generated Files]
        BUILD --> ERROR_HANDLER[Error Handler]
    end

    subgraph "Hot Reload Flow"
        FS -.->|file changes| WATCHER
        WATCHER -.->|triggers| SCANNER
        SCANNER -.->|updates| REG
        REG -.->|notifies| WS
        WS -.->|broadcasts| BROWSER
    end

    classDef discovery fill:#e8f5e8
    classDef server fill:#fff3e0
    classDef build fill:#f3e5f5
    classDef hotreload fill:#e1f5fe,stroke-dasharray: 5 5

    class FS,SCANNER,REG,CACHE discovery
    class REG,SERVER,WS,BROWSER server
    class BUILD,TEMPL,OUTPUT,ERROR_HANDLER build
```

## Key Workflows

### 1. Project Initialization Workflow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant InitService
    participant ConfigBuilder
    participant FileSystem
    participant Registry

    User->>CLI: templar init
    CLI->>InitService: Initialize project
    InitService->>ConfigBuilder: Create default config
    ConfigBuilder->>FileSystem: Write .templar.yml
    InitService->>FileSystem: Create directory structure
    InitService->>FileSystem: Generate example components
    InitService->>Registry: Register components
    Registry-->>CLI: Initialization complete
    CLI-->>User: Project ready
```

### 2. Development Server Startup Workflow

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant ServeService
    participant Scanner
    participant Registry
    participant Server
    participant WebSocket
    participant Browser

    User->>CLI: templar serve
    CLI->>ServeService: Start server
    ServeService->>Scanner: Scan components
    Scanner->>Registry: Register components
    ServeService->>Server: Start HTTP server
    Server->>WebSocket: Initialize WebSocket
    Server-->>CLI: Server started
    CLI-->>User: Server running at :8080
    
    User->>Browser: Open localhost:8080
    Browser->>Server: HTTP Request
    Server->>Registry: Get components
    Registry-->>Server: Component list
    Server-->>Browser: Render web interface
    Browser->>WebSocket: Establish connection
    WebSocket-->>Browser: Connection established
```

### 3. Hot Reload Workflow

```mermaid
sequenceDiagram
    participant Developer
    participant FileSystem
    participant Watcher
    participant Scanner
    participant Registry
    participant BuildPipeline
    participant WebSocket
    participant Browser

    Developer->>FileSystem: Modify component file
    FileSystem->>Watcher: File change event
    Watcher->>Scanner: Trigger rescan
    Scanner->>FileSystem: Read updated file
    Scanner->>Scanner: Parse component
    Scanner->>Registry: Update component
    Registry->>BuildPipeline: Trigger build
    BuildPipeline->>BuildPipeline: Process component
    
    alt Build Success
        BuildPipeline->>WebSocket: Broadcast update
        WebSocket->>Browser: Send reload message
        Browser->>Browser: Refresh component
    else Build Error
        BuildPipeline->>WebSocket: Broadcast error
        WebSocket->>Browser: Show error overlay
    end
```

### 4. Component Build Workflow

```mermaid
sequenceDiagram
    participant CLI
    participant BuildService
    participant Scanner
    participant Registry
    participant Pipeline
    parameter Workers as Worker Pool
    participant TemplCompiler
    participant Cache
    participant ErrorHandler

    CLI->>BuildService: Build components
    BuildService->>Scanner: Scan for components
    Scanner->>Registry: Register components
    BuildService->>Pipeline: Start build
    Pipeline->>Workers: Distribute tasks
    
    par Worker Processing
        Workers->>Cache: Check cache
        alt Cache Hit
            Cache-->>Workers: Return cached result
        else Cache Miss
            Workers->>TemplCompiler: Compile component
            TemplCompiler-->>Workers: Compilation result
            Workers->>Cache: Store result
        end
    end
    
    Workers->>Pipeline: Collect results
    
    alt All Success
        Pipeline-->>BuildService: Build complete
        BuildService-->>CLI: Success
    else Errors Occurred
        Pipeline->>ErrorHandler: Process errors
        ErrorHandler-->>BuildService: Error report
        BuildService-->>CLI: Build failed
    end
```

## Data Flow Diagrams

### Component Discovery Data Flow

```mermaid
flowchart TD
    A[File System Scan] --> B{File Filter}
    B -->|*.templ| C[AST Parser]
    B -->|Other| D[Skip]
    
    C --> E[Extract Metadata]
    E --> F[Component Info]
    F --> G[Hash Generator]
    G --> H[Change Detection]
    
    H --> I{Changed?}
    I -->|Yes| J[Update Registry]
    I -->|No| K[Skip Update]
    
    J --> L[Event Broadcast]
    L --> M[WebSocket Clients]
    L --> N[Build Pipeline]
    
    style A fill:#e8f5e8
    style C fill:#fff3e0
    style J fill:#f3e5f5
    style L fill:#e1f5fe
```

### WebSocket Communication Data Flow

```mermaid
flowchart LR
    subgraph "Server Side"
        A[Registry Events] --> B[WebSocket Hub]
        C[Build Results] --> B
        D[Error Events] --> B
        
        B --> E[Message Queue]
        E --> F[Connection Pool]
        F --> G[Rate Limiter]
        G --> H[Origin Validator]
    end
    
    subgraph "Client Side"
        I[Browser WebSocket] --> J[Message Handler]
        J --> K{Message Type}
        K -->|component_update| L[Refresh View]
        K -->|build_error| M[Show Error]
        K -->|registry_change| N[Update List]
    end
    
    H -->|Valid| I
    L --> O[DOM Update]
    M --> P[Error Overlay]
    N --> Q[Component List]
    
    style B fill:#fff3e0
    style G fill:#fce4ec
    style H fill:#fce4ec
    style J fill:#e1f5fe
```

## Security Architecture

```mermaid
graph TB
    subgraph "Input Validation Layer"
        IV[Input Validator]
        UV[URL Validator]
        PV[Path Validator]
        AV[Argument Validator]
    end
    
    subgraph "Authentication & Authorization"
        AUTH[Authentication]
        AUTHZ[Authorization]
        SESSION[Session Management]
    end
    
    subgraph "Network Security"
        CORS[CORS Handler]
        CSP[CSP Headers]
        RL[Rate Limiting]
        OV[Origin Validation]
    end
    
    subgraph "Command Security"
        CI[Command Injection Prevention]
        AL[Allowlisting]
        SAND[Sandboxing]
    end
    
    subgraph "Data Security"
        PT[Path Traversal Protection]
        ENC[Data Encryption]
        HASH[Secure Hashing]
    end
    
    IV --> AUTH
    UV --> OV
    PV --> PT
    AV --> CI
    
    AUTH --> CORS
    AUTHZ --> SESSION
    
    CORS --> RL
    CSP --> RL
    OV --> RL
    
    CI --> AL
    AL --> SAND
    
    PT --> ENC
    ENC --> HASH
    
    classDef inputLayer fill:#ffebee
    classDef authLayer fill:#fce4ec
    classDef networkLayer fill:#e8eaf6
    classDef commandLayer fill:#e0f2f1
    classDef dataLayer fill:#fff3e0
    
    class IV,UV,PV,AV inputLayer
    class AUTH,AUTHZ,SESSION authLayer
    class CORS,CSP,RL,OV networkLayer
    class CI,AL,SAND commandLayer
    class PT,ENC,HASH dataLayer
```

## Performance Architecture

```mermaid
graph TB
    subgraph "Caching Layer"
        L1[L1: Memory Cache]
        L2[L2: File System Cache]
        L3[L3: Build Cache]
    end
    
    subgraph "Concurrency Layer"
        WP[Worker Pools]
        LF[Lock-Free Operations]
        AP[Async Processing]
    end
    
    subgraph "Optimization Layer"
        OP[Object Pooling]
        MM[Memory Mapping]
        HP[Hash Optimization]
    end
    
    subgraph "Monitoring Layer"
        PM[Performance Monitor]
        BM[Benchmark System]
        RD[Regression Detection]
    end
    
    L1 --> L2
    L2 --> L3
    
    WP --> LF
    LF --> AP
    
    OP --> MM
    MM --> HP
    
    PM --> BM
    BM --> RD
    
    L1 -.-> PM
    WP -.-> PM
    OP -.-> PM
    
    classDef cache fill:#e8f5e8
    classDef concurrency fill:#fff3e0
    classDef optimization fill:#f3e5f5
    classDef monitoring fill:#e1f5fe
    
    class L1,L2,L3 cache
    class WP,LF,AP concurrency
    class OP,MM,HP optimization
    class PM,BM,RD monitoring
```

## Deployment Architecture

```mermaid
graph TB
    subgraph "Development Environment"
        DEV[Developer Machine]
        NIX[Nix Flake Environment]
        LOCAL[Local Testing]
    end
    
    subgraph "CI/CD Pipeline"
        GH[GitHub Actions]
        BUILD[Build Pipeline]
        TEST[Test Suite]
        BENCH[Benchmarks]
    end
    
    subgraph "Container Environment"
        DOCKER[Docker Image]
        COMPOSE[Docker Compose]
        K8S[Kubernetes]
    end
    
    subgraph "Cloud Deployment"
        AWS[AWS ECS]
        GCP[Google Cloud Run]
        AZURE[Azure Container Instances]
    end
    
    subgraph "Monitoring & Observability"
        LOGS[Centralized Logging]
        METRICS[Metrics Collection]
        ALERTS[Alerting System]
        HEALTH[Health Checks]
    end
    
    DEV --> NIX
    NIX --> LOCAL
    LOCAL --> GH
    
    GH --> BUILD
    BUILD --> TEST
    TEST --> BENCH
    BENCH --> DOCKER
    
    DOCKER --> COMPOSE
    DOCKER --> K8S
    DOCKER --> AWS
    DOCKER --> GCP
    DOCKER --> AZURE
    
    AWS --> LOGS
    GCP --> METRICS
    AZURE --> ALERTS
    K8S --> HEALTH
    
    classDef dev fill:#e8f5e8
    classDef cicd fill:#fff3e0
    classDef container fill:#f3e5f5
    classDef cloud fill:#e1f5fe
    classDef monitoring fill:#fce4ec
    
    class DEV,NIX,LOCAL dev
    class GH,BUILD,TEST,BENCH cicd
    class DOCKER,COMPOSE,K8S container
    class AWS,GCP,AZURE cloud
    class LOGS,METRICS,ALERTS,HEALTH monitoring
```

## Architecture Principles

### Interface-Driven Design

The system follows interface-driven design principles with clean abstractions:

- **ComponentRegistry**: Thread-safe component management with change notifications
- **ComponentScanner**: Parallel component discovery and analysis
- **BuildPipeline**: Configurable worker pools with caching and error handling
- **FileFilter**: Flexible file filtering for scanning and watching

### Event-Driven Architecture

Components communicate through events to maintain loose coupling:

- Registry broadcasts component changes
- File watcher triggers scanner updates
- Build pipeline emits completion events
- WebSocket manager distributes real-time updates

### Security-First Design

Security is built into every layer:

- Input validation at all entry points
- Command injection prevention with strict allowlisting
- Path traversal protection with directory enforcement
- WebSocket origin validation with CSRF protection
- Rate limiting and connection management

### Performance Optimization

The system is optimized for high performance:

- **30M+ operations/second** with concurrent processing
- **Object pooling** for memory optimization
- **Lock-free operations** where possible
- **Multi-level caching** with LRU eviction
- **Memory mapping** for large file operations

## Maintenance and Updates

### Diagram Maintenance Process

1. **Automatic Updates**: Diagrams should be updated when major architectural changes occur
2. **Review Process**: All diagram changes require architectural review
3. **Version Control**: Diagrams are version controlled alongside code
4. **Documentation Sync**: Keep diagrams synchronized with implementation
5. **Regular Audits**: Quarterly reviews to ensure diagrams reflect current architecture

### Tools and Standards

- **Mermaid**: Primary diagramming tool for consistency and maintainability
- **GitHub Integration**: Diagrams render automatically in GitHub
- **PlantUML Alternative**: Fallback option for complex diagrams
- **Architecture Decision Records**: Document architectural changes

### Contributing to Architecture

When making architectural changes:

1. Update relevant diagrams first
2. Document the change rationale
3. Review impact on existing components
4. Update this documentation
5. Ensure zero technical debt policy compliance

---

*This documentation is maintained as part of the zero technical debt policy. Last updated: 2025-07-29*