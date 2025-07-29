# Templar API

REST API for the Templar component development server providing component management, rendering, and development tools.

**Version:** 1.0.0

## Servers

- **http://localhost:8080**: Development server
- **https://api.templar.dev**: Production server

## API Endpoints

### /handleeditorindex

#### GET

**Responses:**

- **200**: Success

### /health

#### GET

**Summary:** handleHealth returns the server health status for health checks.

**Responses:**

- **200**: Success

### /api/playground/render

#### POST

**Summary:** handlePlaygroundRender handles interactive component playground rendering.

**Responses:**

- **200**: Success

### /web-socket-enhanced

#### GET

**Summary:** Enhanced WebSocket handler that integrates with existing server.

**Responses:**

- **200**: Success

### /status-endpoint

#### GET

**Summary:** handleStatusEndpoint handles HTTP requests for WebSocket status.

**Responses:**

- **200**: Success

### /handlerender

#### GET

**Responses:**

- **200**: Success

### /handleinlineeditor

#### GET

**Responses:**

- **200**: Success

### /handlebuildmetrics

#### GET

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

### /handleenhancedindex

#### GET

**Responses:**

- **200**: Success

### /handlewebsocket

#### GET

**Responses:**

- **200**: Success

### /enhanced-index

#### GET

**Summary:** handleEnhancedIndex serves the enhanced main interface.

**Responses:**

- **200**: Success

### /playground-component

#### GET

**Summary:** handlePlaygroundComponent serves the interactive playground UI.

**Responses:**

- **200**: Success

### /playground

#### GET

**Summary:** handlePlaygroundIndex serves the main playground page with component list.

**Responses:**

- **200**: Success

### /offline-mode-endpoint

#### GET

**Summary:** handleOfflineModeEndpoint handles offline mode management via HTTP.

**Responses:**

- **200**: Success

### /handlehealth

#### GET

**Responses:**

- **200**: Success

### /handlebuilderrors

#### GET

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

### /handleindex

#### GET

**Responses:**

- **200**: Success

### /index

#### GET

**Responses:**

- **200**: Success

### /render/{name}

#### GET

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| name | string | path | Yes | The name identifier |

**Responses:**

- **200**: Success

### /apispec

#### GET

**Summary:** handleAPISpec serves the OpenAPI specification.

**Responses:**

- **200**: Success

### /playground-index-page

#### GET

**Summary:** handlePlaygroundIndexPage handles playground index page.

**Responses:**

- **200**: Success

### /editor-interface

#### GET

**Summary:** handleEditorInterface handles editor interface requests.

**Responses:**

- **200**: Success

### /editor

#### GET

**Summary:** handleEditorIndex serves the main editor interface.

**Responses:**

- **200**: Success

### /handlecomponent

#### GET

**Responses:**

- **200**: Success

### /handlestatic

#### GET

**Responses:**

- **200**: Success

### /handleeditorapi

#### GET

**Responses:**

- **200**: Success

### /handlebuildcache

#### GET

**Responses:**

- **200**: Success

### /component/{name}

#### GET

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| name | string | path | Yes | The name identifier |

**Responses:**

- **200**: Success

### /renderfileselection

#### GET

**Responses:**

- **200**: Success

### /apidocs

#### GET

**Summary:** handleAPIDocs serves the API documentation interface.

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

### /ws

#### GET

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

### /handlecomponents

#### GET

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

### /handleplaygroundindex

#### GET

**Responses:**

- **200**: Success

### /serveapidocumentation

#### GET

**Summary:** serveAPIDocumentation serves the interactive API documentation interface.

**Responses:**

- **200**: Success

### /serveopenapiyaml

#### GET

**Summary:** serveOpenAPIYAML serves the OpenAPI specification in YAML format.

**Responses:**

- **200**: Success

### /apiversions

#### GET

**Summary:** handleAPIVersions serves available API versions.

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

### /handleplaygroundcomponent

#### GET

**Responses:**

- **200**: Success

### /api/build/cache

#### GET

**Summary:** handleBuildCache manages the build cache.

**Responses:**

- **200**: Success

### /health-endpoint

#### GET

**Summary:** handleHealthEndpoint handles HTTP requests for connection health.

**Responses:**

- **200**: Success

### /handlebuildstatus

#### GET

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

### /serveopenapijson

#### GET

**Summary:** serveOpenAPIJSON serves the OpenAPI specification in JSON format.

**Responses:**

- **200**: Success

### /static-files

#### GET

**Summary:** handleStaticFiles handles static file requests.

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

### /api/build/errors

#### GET

**Summary:** handleBuildErrors returns the last build errors.

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

### /component-editor

#### GET

**Summary:** handleComponentEditor handles the enhanced component editor interface.

**Responses:**

- **200**: Success

### /handleplaygroundrender

#### GET

**Responses:**

- **200**: Success

### /handlefileapi

#### GET

**Responses:**

- **200**: Success

### /handletargetfiles

#### GET

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

### /components

#### GET

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

### /target-files

#### GET

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

### /multiple-files

#### GET

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

### /api/build/status

#### GET

**Summary:** handleBuildStatus returns the current build status.

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

### /api/build/metrics

#### GET

**Summary:** handleBuildMetrics returns detailed build metrics.

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

### /inline-editor

#### GET

**Summary:** handleInlineEditor handles inline editor requests.

**Responses:**

- **200**: Success

### /static/{path}

#### GET

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| path | string | path | Yes | The path identifier |

**Responses:**

- **200**: Success

### /api/editor

#### POST

**Summary:** handleEditorAPI handles editor API requests.

**Responses:**

- **200**: Success

### /api/files

#### GET

**Summary:** handleFileAPI handles file management API requests.

**Parameters:**

| Name | Type | In | Required | Description |
|------|------|----|---------|--------------|
| limit | integer | query | No | Maximum number of items to return |
| offset | integer | query | No | Number of items to skip |

**Responses:**

- **200**: Success

## Data Models

### BuildStatus

Build system status information

**Properties:**

| Name | Type | Required | Description |
|------|------|----------|--------------|
| status | string | Yes | Build status |
| total_builds | integer | No | Total number of builds executed |
| failed_builds | integer | No | Number of failed builds |
| cache_hits | integer | No | Number of cache hits |
| errors | integer | No | Number of current errors |
| timestamp | integer | Yes | Status timestamp (Unix timestamp) |

### ComponentInfo

Comprehensive metadata about a discovered templ component

**Properties:**

| Name | Type | Required | Description |
|------|------|----------|--------------|
| name | string | Yes | The component identifier (e.g., 'Button', 'CardHeader') |
| package | string | Yes | The Go package name where the component is defined |
| imports | array | No | Go packages imported by the component template |
| lastMod | string | No | Last modification time for change detection |
| isExported | boolean | No | Indicates if the component function is exported (public) |
| isRenderable | boolean | No | Indicates if the component can be rendered independently |
| description | string | No | Human-readable documentation for the component |
| filePath | string | Yes | The absolute path to the .templ file containing the component |
| parameters | array | No | Component input parameters and their types |
| hash | string | No | CRC32 checksum for efficient change detection |
| dependencies | array | No | Other components or files this component depends on |
| metadata | object | No | Plugin-specific or custom component information |
| examples | array | No | Sample usage scenarios for the component |

### ParameterInfo

Describes a component parameter extracted from the templ function signature

**Properties:**

| Name | Type | Required | Description |
|------|------|----------|--------------|
| default |  | No | The default value if one is specified (may be null) |
| description | string | No | Documentation for the parameter |
| name | string | Yes | The parameter name as declared in the templ function |
| type | string | Yes | The Go type of the parameter (e.g., 'string', '*User', '[]Item') |
| optional | boolean | No | Indicates if the parameter has a default value or is pointer type |

### ComponentExample

Represents a usage example for a component

**Properties:**

| Name | Type | Required | Description |
|------|------|----------|--------------|
| description | string | Yes | Explains what this example demonstrates |
| props | object | No | The example parameter values |
| code | string | No | The example templ code |
| name | string | Yes | The example identifier |

### Error

Standard error response

**Properties:**

| Name | Type | Required | Description |
|------|------|----------|--------------|
| error | string | Yes | Error message |
| code | integer | No | Error code |
| details | string | No | Additional error details |

### HealthStatus

Server health status information

**Properties:**

| Name | Type | Required | Description |
|------|------|----------|--------------|
| status | string | Yes | Overall health status |
| timestamp | string | Yes | Health check timestamp |
| version | string | No | Application version |
| checks | object | No | Individual component health checks |

