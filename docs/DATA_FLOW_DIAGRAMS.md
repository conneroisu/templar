# Data Flow Diagrams Documentation

This document provides comprehensive data flow diagrams showing how data moves through the Templar system, including data transformations, storage, and processing flows.

## Table of Contents

- [Overview](#overview)
- [High-Level Data Flow](#high-level-data-flow)
- [Component Data Processing Flow](#component-data-processing-flow)
- [Build Pipeline Data Flow](#build-pipeline-data-flow)
- [WebSocket Data Flow](#websocket-data-flow)
- [Configuration Data Flow](#configuration-data-flow)
- [Error Data Flow](#error-data-flow)
- [Performance Data Flow](#performance-data-flow)
- [Security Data Flow](#security-data-flow)

## Overview

Data flow diagrams (DFDs) show how data moves through the Templar system, illustrating data sources, transformations, storage, and destinations. These diagrams help understand data lifecycle and identify potential bottlenecks or security concerns.

## High-Level Data Flow

```mermaid
flowchart TD
    subgraph "Data Sources"
        FS[File System]
        CLI[CLI Input]
        WEB[Web Requests]
        CONFIG[Configuration Files]
    end
    
    subgraph "Data Processing Layer"
        PARSER[Component Parser]
        VALIDATOR[Input Validator]
        TRANSFORMER[Data Transformer]
        BUILDER[Build Processor]
    end
    
    subgraph "Data Storage Layer"
        REGISTRY[Component Registry]
        CACHE[Build Cache]
        MEMORY[Memory Store]
        METRICS[Metrics Store]
    end
    
    subgraph "Data Output Layer"
        HTTP[HTTP Responses]
        WS[WebSocket Messages]
        FILES[Generated Files]
        LOGS[Log Files]
    end
    
    subgraph "Data Flow Processes"
        P1[Component Discovery]
        P2[Build Processing]
        P3[Real-time Updates]
        P4[Performance Monitoring]
    end
    
    %% Input flows
    FS -->|Component Files| PARSER
    CLI -->|Commands & Flags| VALIDATOR
    WEB -->|HTTP Requests| VALIDATOR
    CONFIG -->|Configuration Data| TRANSFORMER
    
    %% Processing flows
    PARSER -->|Parsed Components| P1
    VALIDATOR -->|Validated Input| P2
    TRANSFORMER -->|Config Objects| P1
    BUILDER -->|Build Results| P2
    
    %% Storage flows
    P1 -->|Component Metadata| REGISTRY
    P2 -->|Build Artifacts| CACHE
    P3 -->|Real-time Data| MEMORY
    P4 -->|Performance Data| METRICS
    
    %% Output flows
    REGISTRY -->|Component Lists| HTTP
    CACHE -->|Build Status| WS
    MEMORY -->|Live Updates| WS
    METRICS -->|Performance Stats| LOGS
    
    P2 -->|Generated Code| FILES
    P3 -->|Event Data| WS
    P4 -->|Metrics Data| HTTP
    
    classDef source fill:#e8f5e8
    classDef processing fill:#fff3e0
    classDef storage fill:#f3e5f5
    classDef output fill:#e1f5fe
    classDef process fill:#fce4ec
    
    class FS,CLI,WEB,CONFIG source
    class PARSER,VALIDATOR,TRANSFORMER,BUILDER processing
    class REGISTRY,CACHE,MEMORY,METRICS storage
    class HTTP,WS,FILES,LOGS output
    class P1,P2,P3,P4 process
```

## Component Data Processing Flow

```mermaid
flowchart TD
    subgraph "File System Input"
        TEMPL_FILES[*.templ Files]
        CONFIG_FILES[.templar.yml]
        STATIC_FILES[Static Assets]
    end
    
    subgraph "Parsing & Analysis"
        FILE_READER[File Reader]
        AST_PARSER[AST Parser]
        METADATA_EXTRACTOR[Metadata Extractor]
        DEPENDENCY_ANALYZER[Dependency Analyzer]
        HASH_CALCULATOR[Hash Calculator]
    end
    
    subgraph "Data Transformation"
        COMPONENT_BUILDER[Component Builder]
        PARAMETER_PROCESSOR[Parameter Processor]
        IMPORT_RESOLVER[Import Resolver]
        TYPE_ANALYZER[Type Analyzer]
    end
    
    subgraph "Validation & Security"
        SYNTAX_VALIDATOR[Syntax Validator]
        SECURITY_CHECKER[Security Checker]
        PATH_VALIDATOR[Path Validator]
        CONTENT_SANITIZER[Content Sanitizer]
    end
    
    subgraph "Storage & Indexing"
        COMPONENT_STORE[Component Store]
        DEPENDENCY_INDEX[Dependency Index]
        SEARCH_INDEX[Search Index]
        CACHE_STORE[Cache Store]
    end
    
    subgraph "Output Generation"
        COMPONENT_INFO[Component Info Objects]
        DEPENDENCY_GRAPH[Dependency Graph]
        API_RESPONSES[API Response Data]
        ERROR_REPORTS[Error Reports]
    end
    
    %% Input processing
    TEMPL_FILES --> FILE_READER
    CONFIG_FILES --> FILE_READER
    STATIC_FILES --> FILE_READER
    
    %% Parsing chain
    FILE_READER -->|Raw Content| AST_PARSER
    AST_PARSER -->|AST Nodes| METADATA_EXTRACTOR
    AST_PARSER -->|AST Structure| DEPENDENCY_ANALYZER
    FILE_READER -->|File Content| HASH_CALCULATOR
    
    %% Transformation chain
    METADATA_EXTRACTOR -->|Component Metadata| COMPONENT_BUILDER
    METADATA_EXTRACTOR -->|Parameter Info| PARAMETER_PROCESSOR
    DEPENDENCY_ANALYZER -->|Import Statements| IMPORT_RESOLVER
    COMPONENT_BUILDER -->|Component Data| TYPE_ANALYZER
    
    %% Validation chain
    AST_PARSER -->|Syntax Tree| SYNTAX_VALIDATOR
    COMPONENT_BUILDER -->|Component Data| SECURITY_CHECKER
    FILE_READER -->|File Paths| PATH_VALIDATOR
    METADATA_EXTRACTOR -->|Content| CONTENT_SANITIZER
    
    %% Storage operations
    COMPONENT_BUILDER -->|Component Objects| COMPONENT_STORE
    DEPENDENCY_ANALYZER -->|Dependencies| DEPENDENCY_INDEX
    METADATA_EXTRACTOR -->|Searchable Data| SEARCH_INDEX
    HASH_CALCULATOR -->|Hashes| CACHE_STORE
    
    %% Output generation
    COMPONENT_STORE -->|Stored Components| COMPONENT_INFO
    DEPENDENCY_INDEX -->|Dependency Data| DEPENDENCY_GRAPH
    COMPONENT_INFO -->|Formatted Data| API_RESPONSES
    SYNTAX_VALIDATOR -->|Validation Errors| ERROR_REPORTS
    SECURITY_CHECKER -->|Security Issues| ERROR_REPORTS
    
    classDef input fill:#e8f5e8
    classDef parsing fill:#fff3e0
    classDef transform fill:#f3e5f5
    classDef validate fill:#ffebee
    classDef storage fill:#e1f5fe
    classDef output fill:#fce4ec
    
    class TEMPL_FILES,CONFIG_FILES,STATIC_FILES input
    class FILE_READER,AST_PARSER,METADATA_EXTRACTOR,DEPENDENCY_ANALYZER,HASH_CALCULATOR parsing
    class COMPONENT_BUILDER,PARAMETER_PROCESSOR,IMPORT_RESOLVER,TYPE_ANALYZER transform
    class SYNTAX_VALIDATOR,SECURITY_CHECKER,PATH_VALIDATOR,CONTENT_SANITIZER validate
    class COMPONENT_STORE,DEPENDENCY_INDEX,SEARCH_INDEX,CACHE_STORE storage
    class COMPONENT_INFO,DEPENDENCY_GRAPH,API_RESPONSES,ERROR_REPORTS output
```

## Build Pipeline Data Flow

```mermaid
flowchart TD
    subgraph "Build Input Sources"
        COMPONENTS[Component Registry]
        TEMPLATES[Template Files]
        ASSETS[Static Assets]
        BUILD_CONFIG[Build Configuration]
    end
    
    subgraph "Task Management"
        TASK_QUEUE[Task Queue]
        PRIORITY_SORTER[Priority Sorter]
        DEPENDENCY_RESOLVER[Dependency Resolver]
        TASK_DISTRIBUTOR[Task Distributor]
    end
    
    subgraph "Worker Pool Processing"
        W1[Worker 1]
        W2[Worker 2]
        W3[Worker 3]
        WN[Worker N]
        WORKER_COORDINATOR[Worker Coordinator]
    end
    
    subgraph "Compilation Pipeline"
        TEMPLATE_COMPILER[Template Compiler]
        CODE_GENERATOR[Code Generator]
        ASSET_PROCESSOR[Asset Processor]
        MINIFIER[Minifier]
    end
    
    subgraph "Caching System"
        L1_CACHE[L1 Memory Cache]
        L2_CACHE[L2 Disk Cache]
        CACHE_MANAGER[Cache Manager]
        INVALIDATION_TRACKER[Invalidation Tracker]
    end
    
    subgraph "Output Processing"
        RESULT_COLLECTOR[Result Collector]
        ERROR_AGGREGATOR[Error Aggregator]
        OUTPUT_FORMATTER[Output Formatter]
        FILE_WRITER[File Writer]
    end
    
    subgraph "Build Outputs"
        GENERATED_GO[Generated Go Files]
        COMPILED_ASSETS[Compiled Assets]
        BUILD_MANIFEST[Build Manifest]
        ERROR_REPORTS[Build Error Reports]
        METRICS_DATA[Build Metrics]
    end
    
    %% Input flow
    COMPONENTS -->|Component List| TASK_QUEUE
    TEMPLATES -->|Template Data| TASK_QUEUE
    ASSETS -->|Asset Files| TASK_QUEUE
    BUILD_CONFIG -->|Build Rules| TASK_QUEUE
    
    %% Task management flow
    TASK_QUEUE -->|Raw Tasks| PRIORITY_SORTER
    PRIORITY_SORTER -->|Prioritized Tasks| DEPENDENCY_RESOLVER
    DEPENDENCY_RESOLVER -->|Resolved Dependencies| TASK_DISTRIBUTOR
    TASK_DISTRIBUTOR -->|Distributed Tasks| WORKER_COORDINATOR
    
    %% Worker processing flow
    WORKER_COORDINATOR -->|Task Assignment| W1
    WORKER_COORDINATOR -->|Task Assignment| W2
    WORKER_COORDINATOR -->|Task Assignment| W3
    WORKER_COORDINATOR -->|Task Assignment| WN
    
    %% Compilation flow
    W1 -->|Template Tasks| TEMPLATE_COMPILER
    W2 -->|Code Gen Tasks| CODE_GENERATOR
    W3 -->|Asset Tasks| ASSET_PROCESSOR
    WN -->|Minify Tasks| MINIFIER
    
    TEMPLATE_COMPILER -->|Generated Code| CODE_GENERATOR
    ASSET_PROCESSOR -->|Processed Assets| MINIFIER
    
    %% Caching flow
    TEMPLATE_COMPILER -.->|Cache Check| CACHE_MANAGER
    CODE_GENERATOR -.->|Cache Check| CACHE_MANAGER
    CACHE_MANAGER -.->|L1 Hit| L1_CACHE
    CACHE_MANAGER -.->|L2 Hit| L2_CACHE
    CACHE_MANAGER -.->|Cache Store| L1_CACHE
    CACHE_MANAGER -.->|Cache Store| L2_CACHE
    INVALIDATION_TRACKER -.->|Invalidation| CACHE_MANAGER
    
    %% Output collection flow
    CODE_GENERATOR -->|Generated Files| RESULT_COLLECTOR
    MINIFIER -->|Optimized Assets| RESULT_COLLECTOR
    W1 -->|Worker Errors| ERROR_AGGREGATOR
    W2 -->|Worker Errors| ERROR_AGGREGATOR
    W3 -->|Worker Errors| ERROR_AGGREGATOR
    WN -->|Worker Errors| ERROR_AGGREGATOR
    
    %% Final output flow
    RESULT_COLLECTOR -->|Build Results| OUTPUT_FORMATTER
    ERROR_AGGREGATOR -->|Collected Errors| OUTPUT_FORMATTER
    OUTPUT_FORMATTER -->|Formatted Output| FILE_WRITER
    
    FILE_WRITER -->|Go Code| GENERATED_GO
    FILE_WRITER -->|Assets| COMPILED_ASSETS
    FILE_WRITER -->|Manifest| BUILD_MANIFEST
    ERROR_AGGREGATOR -->|Error Data| ERROR_REPORTS
    RESULT_COLLECTOR -->|Performance Data| METRICS_DATA
    
    classDef input fill:#e8f5e8
    classDef task fill:#fff3e0
    classDef worker fill:#f3e5f5
    classDef compile fill:#e1f5fe
    classDef cache fill:#fce4ec,stroke-dasharray: 5 5
    classDef output fill:#ffebee
    classDef final fill:#e0f2f1
    
    class COMPONENTS,TEMPLATES,ASSETS,BUILD_CONFIG input
    class TASK_QUEUE,PRIORITY_SORTER,DEPENDENCY_RESOLVER,TASK_DISTRIBUTOR task
    class W1,W2,W3,WN,WORKER_COORDINATOR worker
    class TEMPLATE_COMPILER,CODE_GENERATOR,ASSET_PROCESSOR,MINIFIER compile
    class L1_CACHE,L2_CACHE,CACHE_MANAGER,INVALIDATION_TRACKER cache
    class RESULT_COLLECTOR,ERROR_AGGREGATOR,OUTPUT_FORMATTER,FILE_WRITER output
    class GENERATED_GO,COMPILED_ASSETS,BUILD_MANIFEST,ERROR_REPORTS,METRICS_DATA final
```

## WebSocket Data Flow

```mermaid
flowchart TD
    subgraph "Event Sources"
        REGISTRY_EVENTS[Registry Events]
        BUILD_EVENTS[Build Events]
        FILE_EVENTS[File System Events]
        ERROR_EVENTS[Error Events]
        USER_EVENTS[User Actions]
    end
    
    subgraph "Event Processing"
        EVENT_ROUTER[Event Router]
        EVENT_FILTER[Event Filter]
        EVENT_TRANSFORMER[Event Transformer]
        EVENT_PRIORITIZER[Event Prioritizer]
    end
    
    subgraph "Message Pipeline"
        MESSAGE_BUILDER[Message Builder]
        MESSAGE_SERIALIZER[Message Serializer]
        MESSAGE_COMPRESSOR[Message Compressor]
        MESSAGE_ENCRYPTOR[Message Encryptor]
    end
    
    subgraph "Connection Management"
        CONNECTION_POOL[Connection Pool]
        SUBSCRIPTION_MANAGER[Subscription Manager]
        RATE_LIMITER[Rate Limiter]
        ORIGIN_VALIDATOR[Origin Validator]
    end
    
    subgraph "Broadcasting System"
        BROADCAST_SCHEDULER[Broadcast Scheduler]
        FANOUT_PROCESSOR[Fanout Processor]
        DELIVERY_TRACKER[Delivery Tracker]
        RETRY_HANDLER[Retry Handler]
    end
    
    subgraph "Client Communication"
        WS_CONNECTIONS[WebSocket Connections]
        CLIENT_BUFFERS[Client Buffers]
        ACK_TRACKER[Acknowledgment Tracker]
        ERROR_HANDLER[Connection Error Handler]
    end
    
    subgraph "Data Persistence"
        MESSAGE_LOG[Message Log]
        DELIVERY_LOG[Delivery Log]
        ERROR_LOG[Error Log]
        METRICS_STORE[Metrics Store]
    end
    
    %% Event ingestion flow
    REGISTRY_EVENTS -->|Component Changes| EVENT_ROUTER
    BUILD_EVENTS -->|Build Results| EVENT_ROUTER
    FILE_EVENTS -->|File Changes| EVENT_ROUTER
    ERROR_EVENTS -->|Error Data| EVENT_ROUTER
    USER_EVENTS -->|User Actions| EVENT_ROUTER
    
    %% Event processing flow
    EVENT_ROUTER -->|Categorized Events| EVENT_FILTER
    EVENT_FILTER -->|Filtered Events| EVENT_TRANSFORMER
    EVENT_TRANSFORMER -->|Formatted Events| EVENT_PRIORITIZER
    EVENT_PRIORITIZER -->|Prioritized Events| MESSAGE_BUILDER
    
    %% Message pipeline flow
    MESSAGE_BUILDER -->|Raw Messages| MESSAGE_SERIALIZER
    MESSAGE_SERIALIZER -->|JSON Messages| MESSAGE_COMPRESSOR
    MESSAGE_COMPRESSOR -->|Compressed Data| MESSAGE_ENCRYPTOR
    MESSAGE_ENCRYPTOR -->|Encrypted Messages| BROADCAST_SCHEDULER
    
    %% Connection management flow
    CONNECTION_POOL -->|Active Connections| SUBSCRIPTION_MANAGER
    SUBSCRIPTION_MANAGER -->|Subscription Data| RATE_LIMITER
    RATE_LIMITER -->|Rate Limited Clients| ORIGIN_VALIDATOR
    ORIGIN_VALIDATOR -->|Validated Clients| BROADCAST_SCHEDULER
    
    %% Broadcasting flow
    BROADCAST_SCHEDULER -->|Scheduled Messages| FANOUT_PROCESSOR
    FANOUT_PROCESSOR -->|Fanned Out Messages| CLIENT_BUFFERS
    CLIENT_BUFFERS -->|Buffered Messages| WS_CONNECTIONS
    WS_CONNECTIONS -->|Delivery Status| DELIVERY_TRACKER
    DELIVERY_TRACKER -->|Failed Deliveries| RETRY_HANDLER
    RETRY_HANDLER -->|Retry Messages| FANOUT_PROCESSOR
    
    %% Client communication flow
    WS_CONNECTIONS -->|Sent Messages| ACK_TRACKER
    WS_CONNECTIONS -->|Connection Errors| ERROR_HANDLER
    ACK_TRACKER -->|Acknowledgments| DELIVERY_TRACKER
    ERROR_HANDLER -->|Error Handling| CONNECTION_POOL
    
    %% Persistence flow
    MESSAGE_BUILDER -->|Message Data| MESSAGE_LOG
    DELIVERY_TRACKER -->|Delivery Data| DELIVERY_LOG
    ERROR_HANDLER -->|Error Data| ERROR_LOG
    FANOUT_PROCESSOR -->|Performance Data| METRICS_STORE
    
    classDef source fill:#e8f5e8
    classDef processing fill:#fff3e0
    classDef pipeline fill:#f3e5f5
    classDef connection fill:#e1f5fe
    classDef broadcast fill:#fce4ec
    classDef client fill:#ffebee
    classDef persistence fill:#e0f2f1
    
    class REGISTRY_EVENTS,BUILD_EVENTS,FILE_EVENTS,ERROR_EVENTS,USER_EVENTS source
    class EVENT_ROUTER,EVENT_FILTER,EVENT_TRANSFORMER,EVENT_PRIORITIZER processing
    class MESSAGE_BUILDER,MESSAGE_SERIALIZER,MESSAGE_COMPRESSOR,MESSAGE_ENCRYPTOR pipeline
    class CONNECTION_POOL,SUBSCRIPTION_MANAGER,RATE_LIMITER,ORIGIN_VALIDATOR connection
    class BROADCAST_SCHEDULER,FANOUT_PROCESSOR,DELIVERY_TRACKER,RETRY_HANDLER broadcast
    class WS_CONNECTIONS,CLIENT_BUFFERS,ACK_TRACKER,ERROR_HANDLER client
    class MESSAGE_LOG,DELIVERY_LOG,ERROR_LOG,METRICS_STORE persistence
```

## Configuration Data Flow

```mermaid
flowchart TD
    subgraph "Configuration Sources"
        DEFAULT_CONFIG[Default Configuration]
        USER_CONFIG[.templar.yml]
        ENV_VARS[Environment Variables]
        CLI_FLAGS[CLI Flags]
        RUNTIME_CONFIG[Runtime Configuration]
    end
    
    subgraph "Configuration Loading"
        CONFIG_LOADER[Configuration Loader]
        FILE_READER[File Reader]
        ENV_READER[Environment Reader]
        FLAG_PARSER[Flag Parser]
        MERGER[Configuration Merger]
    end
    
    subgraph "Validation & Processing"
        SCHEMA_VALIDATOR[Schema Validator]
        TYPE_CONVERTER[Type Converter]
        PATH_RESOLVER[Path Resolver]
        SECURITY_VALIDATOR[Security Validator]
        DEFAULT_APPLIER[Default Applier]
    end
    
    subgraph "Configuration Storage"
        CONFIG_REGISTRY[Configuration Registry]
        SECTION_CACHE[Section Cache]
        WATCH_REGISTRY[Watch Registry]
        VALIDATION_CACHE[Validation Cache]
    end
    
    subgraph "Configuration Distribution"
        CONFIG_PROVIDER[Configuration Provider]
        CHANGE_NOTIFIER[Change Notifier]
        HOT_RELOAD[Hot Reload Manager]
        DEPENDENCY_TRACKER[Dependency Tracker]
    end
    
    subgraph "Configuration Consumers"
        SERVER_CONFIG[Server Configuration]
        BUILD_CONFIG[Build Configuration] 
        SCANNER_CONFIG[Scanner Configuration]
        MONITOR_CONFIG[Monitoring Configuration]
        PLUGIN_CONFIG[Plugin Configuration]
    end
    
    %% Source reading flow
    DEFAULT_CONFIG -->|Default Values| CONFIG_LOADER
    USER_CONFIG -->|User Settings| FILE_READER
    ENV_VARS -->|Environment Settings| ENV_READER
    CLI_FLAGS -->|Command Flags| FLAG_PARSER
    RUNTIME_CONFIG -->|Runtime Settings| CONFIG_LOADER
    
    %% Loading flow
    FILE_READER -->|File Content| CONFIG_LOADER
    ENV_READER -->|Environment Data| CONFIG_LOADER
    FLAG_PARSER -->|Parsed Flags| CONFIG_LOADER
    CONFIG_LOADER -->|Raw Configuration| MERGER
    
    %% Processing flow
    MERGER -->|Merged Config| SCHEMA_VALIDATOR
    SCHEMA_VALIDATOR -->|Valid Schema| TYPE_CONVERTER
    TYPE_CONVERTER -->|Typed Values| PATH_RESOLVER
    PATH_RESOLVER -->|Resolved Paths| SECURITY_VALIDATOR
    SECURITY_VALIDATOR -->|Secure Config| DEFAULT_APPLIER
    DEFAULT_APPLIER -->|Complete Config| CONFIG_REGISTRY
    
    %% Storage flow
    CONFIG_REGISTRY -->|Configuration Data| SECTION_CACHE
    CONFIG_REGISTRY -->|Watch Patterns| WATCH_REGISTRY
    SCHEMA_VALIDATOR -->|Validation Results| VALIDATION_CACHE
    
    %% Distribution flow
    CONFIG_REGISTRY -->|Configuration| CONFIG_PROVIDER
    CONFIG_PROVIDER -->|Config Updates| CHANGE_NOTIFIER
    CHANGE_NOTIFIER -->|Change Events| HOT_RELOAD
    CONFIG_PROVIDER -->|Dependencies| DEPENDENCY_TRACKER
    
    %% Consumer flow
    CONFIG_PROVIDER -->|Server Settings| SERVER_CONFIG
    CONFIG_PROVIDER -->|Build Settings| BUILD_CONFIG
    CONFIG_PROVIDER -->|Scanner Settings| SCANNER_CONFIG
    CONFIG_PROVIDER -->|Monitor Settings| MONITOR_CONFIG
    CONFIG_PROVIDER -->|Plugin Settings| PLUGIN_CONFIG
    
    %% Hot reload flow
    USER_CONFIG -.->|File Changes| WATCH_REGISTRY
    WATCH_REGISTRY -.->|Change Detection| HOT_RELOAD
    HOT_RELOAD -.->|Reload Trigger| CONFIG_LOADER
    
    classDef source fill:#e8f5e8
    classDef loading fill:#fff3e0
    classDef processing fill:#f3e5f5
    classDef storage fill:#e1f5fe
    classDef distribution fill:#fce4ec
    classDef consumer fill:#ffebee
    classDef hotreload fill:#e0f2f1,stroke-dasharray: 5 5
    
    class DEFAULT_CONFIG,USER_CONFIG,ENV_VARS,CLI_FLAGS,RUNTIME_CONFIG source
    class CONFIG_LOADER,FILE_READER,ENV_READER,FLAG_PARSER,MERGER loading
    class SCHEMA_VALIDATOR,TYPE_CONVERTER,PATH_RESOLVER,SECURITY_VALIDATOR,DEFAULT_APPLIER processing
    class CONFIG_REGISTRY,SECTION_CACHE,WATCH_REGISTRY,VALIDATION_CACHE storage
    class CONFIG_PROVIDER,CHANGE_NOTIFIER,HOT_RELOAD,DEPENDENCY_TRACKER distribution
    class SERVER_CONFIG,BUILD_CONFIG,SCANNER_CONFIG,MONITOR_CONFIG,PLUGIN_CONFIG consumer
```

## Error Data Flow

```mermaid
flowchart TD
    subgraph "Error Sources"
        PARSE_ERRORS[Parse Errors]
        BUILD_ERRORS[Build Errors]
        RUNTIME_ERRORS[Runtime Errors]
        VALIDATION_ERRORS[Validation Errors]
        SECURITY_ERRORS[Security Errors]
        NETWORK_ERRORS[Network Errors]
    end
    
    subgraph "Error Capture"
        ERROR_INTERCEPTOR[Error Interceptor]
        PANIC_HANDLER[Panic Handler]
        SIGNAL_HANDLER[Signal Handler]
        EXCEPTION_HANDLER[Exception Handler]
    end
    
    subgraph "Error Processing"
        ERROR_CLASSIFIER[Error Classifier]
        CONTEXT_ENRICHER[Context Enricher]
        STACK_TRACER[Stack Tracer]
        ERROR_AGGREGATOR[Error Aggregator]
        SEVERITY_ANALYZER[Severity Analyzer]
    end
    
    subgraph "Error Storage"
        ERROR_QUEUE[Error Queue]
        ERROR_BUFFER[Error Buffer]
        ERROR_DATABASE[Error Database]
        ERROR_INDEX[Error Index]
    end
    
    subgraph "Error Analysis"
        PATTERN_DETECTOR[Pattern Detector]
        CORRELATION_ENGINE[Correlation Engine]
        TREND_ANALYZER[Trend Analyzer]
        ROOT_CAUSE_ANALYZER[Root Cause Analyzer]
    end
    
    subgraph "Error Reporting"
        ERROR_FORMATTER[Error Formatter]
        REPORT_GENERATOR[Report Generator]
        ALERT_MANAGER[Alert Manager]
        NOTIFICATION_SENDER[Notification Sender]
    end
    
    subgraph "Error Response"
        USER_NOTIFICATIONS[User Notifications]
        DEVELOPER_ALERTS[Developer Alerts]
        SYSTEM_LOGS[System Logs]
        MONITORING_DASHBOARDS[Monitoring Dashboards]
        ERROR_OVERLAYS[Error Overlays]
    end
    
    %% Error capture flow
    PARSE_ERRORS -->|Parse Issues| ERROR_INTERCEPTOR
    BUILD_ERRORS -->|Build Failures| ERROR_INTERCEPTOR
    RUNTIME_ERRORS -->|Runtime Issues| ERROR_INTERCEPTOR
    VALIDATION_ERRORS -->|Validation Issues| ERROR_INTERCEPTOR
    SECURITY_ERRORS -->|Security Issues| ERROR_INTERCEPTOR
    NETWORK_ERRORS -->|Network Issues| ERROR_INTERCEPTOR
    
    %% Specialized capture
    RUNTIME_ERRORS -->|Panics| PANIC_HANDLER
    RUNTIME_ERRORS -->|Signals| SIGNAL_HANDLER
    RUNTIME_ERRORS -->|Exceptions| EXCEPTION_HANDLER
    
    %% Processing flow
    ERROR_INTERCEPTOR -->|Raw Errors| ERROR_CLASSIFIER
    PANIC_HANDLER -->|Panic Data| ERROR_CLASSIFIER
    SIGNAL_HANDLER -->|Signal Data| ERROR_CLASSIFIER
    EXCEPTION_HANDLER -->|Exception Data| ERROR_CLASSIFIER
    
    ERROR_CLASSIFIER -->|Classified Errors| CONTEXT_ENRICHER
    CONTEXT_ENRICHER -->|Enriched Errors| STACK_TRACER
    STACK_TRACER -->|Stack Traces| SEVERITY_ANALYZER
    SEVERITY_ANALYZER -->|Severity Data| ERROR_AGGREGATOR
    
    %% Storage flow
    ERROR_AGGREGATOR -->|Aggregated Errors| ERROR_QUEUE
    ERROR_QUEUE -->|Queued Errors| ERROR_BUFFER
    ERROR_BUFFER -->|Buffered Errors| ERROR_DATABASE
    ERROR_DATABASE -->|Indexed Errors| ERROR_INDEX
    
    %% Analysis flow
    ERROR_DATABASE -->|Error Data| PATTERN_DETECTOR
    ERROR_DATABASE -->|Historical Data| CORRELATION_ENGINE
    ERROR_DATABASE -->|Time Series| TREND_ANALYZER
    PATTERN_DETECTOR -->|Patterns| ROOT_CAUSE_ANALYZER
    CORRELATION_ENGINE -->|Correlations| ROOT_CAUSE_ANALYZER
    TREND_ANALYZER -->|Trends| ROOT_CAUSE_ANALYZER
    
    %% Reporting flow
    ROOT_CAUSE_ANALYZER -->|Analysis Results| ERROR_FORMATTER
    ERROR_AGGREGATOR -->|Current Errors| ERROR_FORMATTER
    ERROR_FORMATTER -->|Formatted Errors| REPORT_GENERATOR
    REPORT_GENERATOR -->|Reports| ALERT_MANAGER
    ALERT_MANAGER -->|Alerts| NOTIFICATION_SENDER
    
    %% Response flow
    NOTIFICATION_SENDER -->|User Messages| USER_NOTIFICATIONS
    NOTIFICATION_SENDER -->|Dev Alerts| DEVELOPER_ALERTS
    ERROR_FORMATTER -->|Log Entries| SYSTEM_LOGS
    REPORT_GENERATOR -->|Dashboard Data| MONITORING_DASHBOARDS
    ERROR_FORMATTER -->|UI Errors| ERROR_OVERLAYS
    
    classDef source fill:#ffebee
    classDef capture fill:#fff3e0
    classDef processing fill:#f3e5f5
    classDef storage fill:#e1f5fe
    classDef analysis fill:#fce4ec
    classDef reporting fill:#e8f5e8
    classDef response fill:#e0f2f1
    
    class PARSE_ERRORS,BUILD_ERRORS,RUNTIME_ERRORS,VALIDATION_ERRORS,SECURITY_ERRORS,NETWORK_ERRORS source
    class ERROR_INTERCEPTOR,PANIC_HANDLER,SIGNAL_HANDLER,EXCEPTION_HANDLER capture
    class ERROR_CLASSIFIER,CONTEXT_ENRICHER,STACK_TRACER,ERROR_AGGREGATOR,SEVERITY_ANALYZER processing
    class ERROR_QUEUE,ERROR_BUFFER,ERROR_DATABASE,ERROR_INDEX storage
    class PATTERN_DETECTOR,CORRELATION_ENGINE,TREND_ANALYZER,ROOT_CAUSE_ANALYZER analysis
    class ERROR_FORMATTER,REPORT_GENERATOR,ALERT_MANAGER,NOTIFICATION_SENDER reporting
    class USER_NOTIFICATIONS,DEVELOPER_ALERTS,SYSTEM_LOGS,MONITORING_DASHBOARDS,ERROR_OVERLAYS response
```

## Performance Data Flow

```mermaid
flowchart TD
    subgraph "Performance Sources"
        OPERATION_METRICS[Operation Metrics]
        SYSTEM_METRICS[System Metrics]
        RESOURCE_METRICS[Resource Metrics]
        NETWORK_METRICS[Network Metrics]
        CUSTOM_METRICS[Custom Metrics]
    end
    
    subgraph "Data Collection"
        METRIC_COLLECTOR[Metric Collector]
        SAMPLING_ENGINE[Sampling Engine]
        AGGREGATION_ENGINE[Aggregation Engine]
        FILTERING_ENGINE[Filtering Engine]
    end
    
    subgraph "Data Processing"
        STATISTICS_PROCESSOR[Statistics Processor]
        PERCENTILE_CALCULATOR[Percentile Calculator]
        TREND_CALCULATOR[Trend Calculator]
        BASELINE_COMPARER[Baseline Comparer]
    end
    
    subgraph "Data Storage"
        TIME_SERIES_DB[Time Series Database]
        METRIC_BUFFER[Metric Buffer]
        BASELINE_STORE[Baseline Store]
        HISTOGRAM_STORE[Histogram Store]
    end
    
    subgraph "Analysis Engine"
        REGRESSION_DETECTOR[Regression Detector]
        ANOMALY_DETECTOR[Anomaly Detector]
        CORRELATION_ANALYZER[Correlation Analyzer]
        PREDICTION_ENGINE[Prediction Engine]
    end
    
    subgraph "Visualization & Reporting"
        DASHBOARD_ENGINE[Dashboard Engine]
        CHART_GENERATOR[Chart Generator]
        REPORT_BUILDER[Report Builder]
        ALERT_GENERATOR[Alert Generator]
    end
    
    subgraph "Performance Outputs"
        REAL_TIME_DASHBOARDS[Real-time Dashboards]
        PERFORMANCE_REPORTS[Performance Reports]
        REGRESSION_ALERTS[Regression Alerts]
        OPTIMIZATION_SUGGESTIONS[Optimization Suggestions]
    end
    
    %% Collection flow
    OPERATION_METRICS -->|Operation Data| METRIC_COLLECTOR
    SYSTEM_METRICS -->|System Data| METRIC_COLLECTOR
    RESOURCE_METRICS -->|Resource Data| METRIC_COLLECTOR
    NETWORK_METRICS -->|Network Data| METRIC_COLLECTOR
    CUSTOM_METRICS -->|Custom Data| METRIC_COLLECTOR
    
    %% Processing flow
    METRIC_COLLECTOR -->|Raw Metrics| SAMPLING_ENGINE
    SAMPLING_ENGINE -->|Sampled Data| AGGREGATION_ENGINE
    AGGREGATION_ENGINE -->|Aggregated Metrics| FILTERING_ENGINE
    FILTERING_ENGINE -->|Filtered Metrics| STATISTICS_PROCESSOR
    
    %% Statistics flow
    STATISTICS_PROCESSOR -->|Statistical Data| PERCENTILE_CALCULATOR
    STATISTICS_PROCESSOR -->|Statistical Data| TREND_CALCULATOR
    STATISTICS_PROCESSOR -->|Statistical Data| BASELINE_COMPARER
    
    %% Storage flow
    AGGREGATION_ENGINE -->|Time Series| TIME_SERIES_DB
    FILTERING_ENGINE -->|Buffered Metrics| METRIC_BUFFER
    BASELINE_COMPARER -->|Baseline Data| BASELINE_STORE
    PERCENTILE_CALCULATOR -->|Histogram Data| HISTOGRAM_STORE
    
    %% Analysis flow
    TIME_SERIES_DB -->|Historical Data| REGRESSION_DETECTOR
    TIME_SERIES_DB -->|Time Series| ANOMALY_DETECTOR
    BASELINE_STORE -->|Baseline Data| REGRESSION_DETECTOR
    HISTOGRAM_STORE -->|Distribution Data| CORRELATION_ANALYZER
    TREND_CALCULATOR -->|Trend Data| PREDICTION_ENGINE
    
    %% Visualization flow
    TIME_SERIES_DB -->|Display Data| DASHBOARD_ENGINE
    PERCENTILE_CALCULATOR -->|Chart Data| CHART_GENERATOR
    STATISTICS_PROCESSOR -->|Report Data| REPORT_BUILDER
    REGRESSION_DETECTOR -->|Alert Data| ALERT_GENERATOR
    
    %% Output flow
    DASHBOARD_ENGINE -->|Dashboard Data| REAL_TIME_DASHBOARDS
    CHART_GENERATOR -->|Chart Data| REAL_TIME_DASHBOARDS
    REPORT_BUILDER -->|Report Data| PERFORMANCE_REPORTS
    ALERT_GENERATOR -->|Alert Data| REGRESSION_ALERTS
    PREDICTION_ENGINE -->|Prediction Data| OPTIMIZATION_SUGGESTIONS
    
    classDef source fill:#e8f5e8
    classDef collection fill:#fff3e0
    classDef processing fill:#f3e5f5
    classDef storage fill:#e1f5fe
    classDef analysis fill:#fce4ec
    classDef visualization fill:#ffebee
    classDef output fill:#e0f2f1
    
    class OPERATION_METRICS,SYSTEM_METRICS,RESOURCE_METRICS,NETWORK_METRICS,CUSTOM_METRICS source
    class METRIC_COLLECTOR,SAMPLING_ENGINE,AGGREGATION_ENGINE,FILTERING_ENGINE collection
    class STATISTICS_PROCESSOR,PERCENTILE_CALCULATOR,TREND_CALCULATOR,BASELINE_COMPARER processing
    class TIME_SERIES_DB,METRIC_BUFFER,BASELINE_STORE,HISTOGRAM_STORE storage
    class REGRESSION_DETECTOR,ANOMALY_DETECTOR,CORRELATION_ANALYZER,PREDICTION_ENGINE analysis
    class DASHBOARD_ENGINE,CHART_GENERATOR,REPORT_BUILDER,ALERT_GENERATOR visualization
    class REAL_TIME_DASHBOARDS,PERFORMANCE_REPORTS,REGRESSION_ALERTS,OPTIMIZATION_SUGGESTIONS output
```

## Security Data Flow

```mermaid
flowchart TD
    subgraph "Security Input Sources"
        USER_INPUT[User Input]
        API_REQUESTS[API Requests]
        FILE_UPLOADS[File Uploads]
        WEBSOCKET_MESSAGES[WebSocket Messages]
        CLI_COMMANDS[CLI Commands]
    end
    
    subgraph "Input Validation"
        INPUT_SANITIZER[Input Sanitizer]
        XSS_DETECTOR[XSS Detector]
        INJECTION_DETECTOR[SQL/Command Injection Detector]
        PATH_VALIDATOR[Path Traversal Validator]
        SIZE_VALIDATOR[Size Validator]
    end
    
    subgraph "Authentication & Authorization"
        AUTH_HANDLER[Authentication Handler]
        TOKEN_VALIDATOR[Token Validator]
        PERMISSION_CHECKER[Permission Checker]
        ROLE_VALIDATOR[Role Validator]
    end
    
    subgraph "Security Processing"
        THREAT_ANALYZER[Threat Analyzer]
        BEHAVIOR_ANALYZER[Behavior Analyzer]
        PATTERN_MATCHER[Pattern Matcher]
        RISK_ASSESSOR[Risk Assessor]
    end
    
    subgraph "Security Storage"
        SECURITY_LOG[Security Log]
        THREAT_DATABASE[Threat Database]
        AUDIT_TRAIL[Audit Trail]
        BLOCKED_IPS[Blocked IPs]
        SECURITY_EVENTS[Security Events]
    end
    
    subgraph "Security Monitoring"
        INTRUSION_DETECTOR[Intrusion Detector]
        ANOMALY_DETECTOR[Anomaly Detector]
        ALERT_SYSTEM[Alert System]
        INCIDENT_TRACKER[Incident Tracker]
    end
    
    subgraph "Security Response"
        AUTOMATIC_BLOCKING[Automatic Blocking]
        RATE_LIMITING[Rate Limiting]
        SECURITY_NOTIFICATIONS[Security Notifications]
        INCIDENT_REPORTS[Incident Reports]
        FORENSIC_DATA[Forensic Data]
    end
    
    %% Input validation flow
    USER_INPUT -->|Raw Input| INPUT_SANITIZER
    API_REQUESTS -->|Request Data| INPUT_SANITIZER
    FILE_UPLOADS -->|File Data| INPUT_SANITIZER
    WEBSOCKET_MESSAGES -->|Message Data| INPUT_SANITIZER
    CLI_COMMANDS -->|Command Data| INPUT_SANITIZER
    
    %% Validation pipeline
    INPUT_SANITIZER -->|Sanitized Input| XSS_DETECTOR
    INPUT_SANITIZER -->|Sanitized Input| INJECTION_DETECTOR
    INPUT_SANITIZER -->|File Paths| PATH_VALIDATOR
    INPUT_SANITIZER -->|Data Size| SIZE_VALIDATOR
    
    %% Authentication flow
    XSS_DETECTOR -->|Validated Input| AUTH_HANDLER
    INJECTION_DETECTOR -->|Validated Input| AUTH_HANDLER
    PATH_VALIDATOR -->|Validated Paths| AUTH_HANDLER
    SIZE_VALIDATOR -->|Size Validated| AUTH_HANDLER
    
    AUTH_HANDLER -->|Auth Tokens| TOKEN_VALIDATOR
    TOKEN_VALIDATOR -->|Valid Tokens| PERMISSION_CHECKER
    PERMISSION_CHECKER -->|Permissions| ROLE_VALIDATOR
    
    %% Security processing flow
    ROLE_VALIDATOR -->|Authorized Requests| THREAT_ANALYZER
    THREAT_ANALYZER -->|Threat Data| BEHAVIOR_ANALYZER
    BEHAVIOR_ANALYZER -->|Behavior Data| PATTERN_MATCHER
    PATTERN_MATCHER -->|Pattern Data| RISK_ASSESSOR
    
    %% Storage flow
    THREAT_ANALYZER -->|Threat Info| SECURITY_LOG
    RISK_ASSESSOR -->|Risk Data| THREAT_DATABASE
    AUTH_HANDLER -->|Auth Events| AUDIT_TRAIL
    PATTERN_MATCHER -->|Blocked IPs| BLOCKED_IPS
    BEHAVIOR_ANALYZER -->|Security Events| SECURITY_EVENTS
    
    %% Monitoring flow
    SECURITY_EVENTS -->|Event Data| INTRUSION_DETECTOR
    BEHAVIOR_ANALYZER -->|Behavior Patterns| ANOMALY_DETECTOR
    THREAT_DATABASE -->|Threat Intelligence| ALERT_SYSTEM
    ALERT_SYSTEM -->|Incidents| INCIDENT_TRACKER
    
    %% Response flow
    INTRUSION_DETECTOR -->|Intrusion Detected| AUTOMATIC_BLOCKING
    ANOMALY_DETECTOR -->|Anomalies| RATE_LIMITING
    ALERT_SYSTEM -->|Security Alerts| SECURITY_NOTIFICATIONS
    INCIDENT_TRACKER -->|Incident Data| INCIDENT_REPORTS
    SECURITY_LOG -->|Log Data| FORENSIC_DATA
    
    %% Feedback loops
    AUTOMATIC_BLOCKING -.->|Blocked IPs| BLOCKED_IPS
    RATE_LIMITING -.->|Rate Data| BEHAVIOR_ANALYZER
    INCIDENT_REPORTS -.->|Incident Intelligence| THREAT_DATABASE
    
    classDef input fill:#ffebee
    classDef validation fill:#fff3e0
    classDef auth fill:#f3e5f5
    classDef processing fill:#e1f5fe
    classDef storage fill:#fce4ec
    classDef monitoring fill:#e8f5e8
    classDef response fill:#e0f2f1
    classDef feedback fill:#f5f5f5,stroke-dasharray: 5 5
    
    class USER_INPUT,API_REQUESTS,FILE_UPLOADS,WEBSOCKET_MESSAGES,CLI_COMMANDS input
    class INPUT_SANITIZER,XSS_DETECTOR,INJECTION_DETECTOR,PATH_VALIDATOR,SIZE_VALIDATOR validation
    class AUTH_HANDLER,TOKEN_VALIDATOR,PERMISSION_CHECKER,ROLE_VALIDATOR auth
    class THREAT_ANALYZER,BEHAVIOR_ANALYZER,PATTERN_MATCHER,RISK_ASSESSOR processing
    class SECURITY_LOG,THREAT_DATABASE,AUDIT_TRAIL,BLOCKED_IPS,SECURITY_EVENTS storage
    class INTRUSION_DETECTOR,ANOMALY_DETECTOR,ALERT_SYSTEM,INCIDENT_TRACKER monitoring
    class AUTOMATIC_BLOCKING,RATE_LIMITING,SECURITY_NOTIFICATIONS,INCIDENT_REPORTS,FORENSIC_DATA response
```

## Data Flow Optimization Guidelines

### 1. Data Flow Performance

- **Minimize Data Transformations**: Reduce unnecessary data conversions
- **Use Streaming**: Process data in streams where possible
- **Implement Caching**: Cache frequently accessed data
- **Optimize Serialization**: Use efficient serialization formats

### 2. Data Flow Security

- **Validate at Boundaries**: Validate data at system boundaries
- **Encrypt Sensitive Data**: Encrypt sensitive data in transit and at rest
- **Audit Data Access**: Log all data access operations
- **Implement Data Loss Prevention**: Prevent sensitive data leakage

### 3. Data Flow Monitoring

- **Track Data Volume**: Monitor data volume and throughput
- **Measure Latency**: Track data processing latency
- **Monitor Quality**: Validate data quality and integrity
- **Alert on Anomalies**: Alert on unusual data patterns

### 4. Data Flow Maintenance

- **Regular Reviews**: Review data flows quarterly
- **Update Documentation**: Keep diagrams current with implementation
- **Performance Testing**: Regularly test data flow performance
- **Capacity Planning**: Plan for data growth and scaling

---

*These data flow diagrams provide comprehensive views of how data moves through the Templar system. They should be updated whenever significant changes are made to data processing or storage. Last updated: 2025-07-29*