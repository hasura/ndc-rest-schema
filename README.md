# NDC REST Schema

This module includes libraries and tools to convert other API schemas to Native Data Connector (NDC) schema, as well as extend the NDC spec with REST request information.

## Features

- Convert API documentation to NDC schema
  - OpenAPI [2.0](https://swagger.io/specification/v2/) (`openapi2`)
  - OpenAPI [3.0](https://swagger.io/specification/v3)/[3.1](https://swagger.io/specification/) (`openapi3`)
- Convert JSON to YAML. It's helpful to convert JSON schema

## Installation

## Build from source

**Prerequisites**

- Go 1.21+

**Install**

```go
go install github.com/hasura/ndc-rest-schema
```

## Quick start

```sh
Usage: ndc-rest-schema <command>

Flags:
  -h, --help                Show context-sensitive help.
      --log-level="info"    Log level.

Commands:
  convert --file=STRING
    Convert API spec to NDC schema. For example:

        ndc-rest-schema convert -f petstore.yaml -o petstore.json

  json2yaml --file=STRING
    Convert JSON file to YAML. For example:

        ndc-rest-schema json2yaml -f petstore.json -o petstore.yaml

  version
    Print the CLI version.
```

Convert an OpenAPI v3 file to NDC schema with the `convert` command. The tool can accept either file path or URL. The output format can be in JSON or YAML, depending on the file extension:

```sh
ndc-rest-schema convert -f https://raw.githubusercontent.com/OAI/OpenAPI-Specification/main/examples/v3.0/petstore.yaml -o petstore.json --spec openapi3
```

The `--spec` flag represents the input specification:

- `openapi3`: OpenAPI 3.0 and 3.1 (default)
- `openapi2`: OpenAPI 2.0

The output schema can extend from NDC schema with REST information that will be used for NDC REST connector. You can convert the pure NDC schema with `--pure` flag.

## NDC REST configuration

### Request

The NDC REST configuration adds `request` information into `functions` and `procedures` so the connector can have more context to initiate HTTP requests to the remote REST service. The request schema is inspired of [OpenAPI 3 paths and operations](https://swagger.io/docs/specification/paths-and-operations/).

```yaml
- request:
    url: /pets/{petId}
    method: get
    type: rest
    headers:
      Foo: bar
    timeout: 30 # seconds, default 30s
    parameters:
      - name: petId
        in: path
        required: true
        schema:
          type: string
    security:
      - api_key: []
```

The URL can be a relative path or absolute URL. If the URL the relative, there must be a base URL in `settings`:

```yaml
settings:
  servers:
    - url: http://petstore.swagger.io/v1
```

`parameters` include the list of URL and query parameters so the connector can replace values from request arguments.

For procedures, the `body` argument is always treated as the request body. If there is a parameter which has the same name, the tool will rename it to `paramBody`.

### Settings

The `settings` object contains global configuration about servers, authentication and other information.

- `servers`: list of servers that serve the API service.
  - `url`: the base URL of the API server.
- `headers`: default headers will be injected into all requests.
- `timeout`: default timeout for all requests
- `securitySchemes`: global configurations for authentication, follow the [security scheme](https://swagger.io/docs/specification/authentication/) of OpenAPI 3.
- `security`: default [authentication requirements](https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.0.md#security-requirement-object) will be applied to all requests.

### Environment variable template

Environment variable template which is in `{{CONSTANT_CASE}}` or `{{CONSTANT_CASE:-some_default_value}}` format can be replaced with value in the runtime. The wrapper should be double-brackets to avoid mistaking with OpenAPI variable template which is single.

### Full example

```yaml
settings:
  servers:
    - url: "{{PET_STORE_SERVER_URL:-https://petstore3.swagger.io/api/v3}}"
  timeout: 30
  headers:
    foo: bar
  securitySchemes:
    api_key:
      type: apiKey
      value: "{{PET_STORE_API_KEY}}"
      in: header
      name: api_key
    petstore_auth:
      type: oauth2
      flows:
        implicit:
          authorizationUrl: https://petstore3.swagger.io/oauth/authorize
          scopes:
            read:pets: read your pets
            write:pets: modify pets in your account
  security:
    - {}
    - petstore_auth:
        - write:pets
        - read:pets
  version: 1.0.18
collections: []
functions:
  - request:
      url: "/pet/findByStatus"
      method: get
      parameters:
        - name: status
          in: query
          required: false
          schema:
            type: String
            enum:
              - available
              - pending
              - sold
      security:
        - petstore_auth:
            - write:pets
            - read:pets
    arguments:
      status:
        description: Status values that need to be considered for filter
        type:
          type: nullable
          underlying_type:
            name: String
            type: named
    description: Finds Pets by status
    name: findPetsByStatus
    result_type:
      element_type:
        name: Pet
        type: named
      type: array
procedures:
  - request:
      url: "/pet"
      method: post
      headers:
        Content-Type: application/json
      security:
        - petstore_auth:
            - write:pets
            - read:pets
    arguments:
      body:
        description: Request body of /pet
        type:
          name: Pet
          type: named
    description: Add a new pet to the store
    name: addPet
    result_type:
      name: Pet
      type: named
```

## Supported specs

### OpenAPI

The tool can parse and convert OpenAPI documentation to NDC functions and procedures via HTTP methods:

- `GET` -> `Function`
- `POST`, `PUT`, `PATCH`, `DELETE` -> `Procedure`

#### Type conversion

- `boolean` -> `Boolean`
- `string` -> `String`
- `integer` -> `Int`
- `number` -> `Float`
- `object` -> Object types
- `anyOf`, `additionalProperties` and others -> `JSON`

> Because NDC schema doesn't support union types it's impossible to convert dynamic schema to a static type. The `JSON` scalar represent as a dynamic JSON field and don't support nested selection.

#### Naming convention

Schema type names are usually matched with the referenced name. Anonymous type names will be generated from the URL path in PascalCase.

If the `operationId` field exists in API operation, it will be used for functions or procedure name. Otherwise the operation name will be generated from URL path with camelCase format:

```sh
{http_method}{url_path_without_slash}

# GET     /users/{id} => getUsersId
# POST    /users      => postUsers
# DELETE  /users      => deleteUsers
```

You can also change the method alias with `--method-alias=KEY=VALUE;...` flag, for example: `--method-alias=post=create;put=update`.

If the URL path has a prefix such as `/api/v1/users`, you can trim that prefix with `--trim-prefix` flag.

#### Authentication

If the OpenAPI definition has authentication (or security), the tool converts them to `settings` object. The schema is similar to [OpenAPI 3.0 authentication](https://swagger.io/docs/specification/authentication/) with extra configuration fields.

**API Keys**

There is an extra `value` field with environment variable template to be able to replaced in runtime. The name of variable is generated from the security scheme key. For example:

```json
{
  "securitySchemes": {
    "api_key": {
      "type": "apiKey",
      "value": "{{API_KEY}}", // the constant case of api_key
      "in": "header",
      "name": "api_key"
    }
  }
}
```

> You can set the prefix for environment variables with `--env-prefix` flag.

**Auth Token**

This is the general authentication for [Basic](https://swagger.io/docs/specification/authentication/basic-authentication), [Bearer](https://swagger.io/docs/specification/authentication/bearer-authentication/) or any token with scheme. The output credential will be the combination of `scheme` and `value`.

```json
{
  "securitySchemes": {
    "bearer_auth": {
      "type": "http",
      "scheme": "bearer",
      "value": "{{BEARER_AUTH_TOKEN}}", // the constant case of bearer_auth + _TOKEN suffix
      "header": "Authentication"
    }
  }
}
```

```
Authentication: Bearer {{BEARER_AUTH_TOKEN}}
```

The environment variable name of `value` field is the constant case of the security scheme key with `_TOKEN` suffix.

> You can set the prefix for environment variables with `--env-prefix` flag.

**OAuth 2.0**

See [OAuth 2.0](https://swagger.io/docs/specification/authentication/oauth2) section of OpenAPI 3.

**OpenID Connect Discovery**

See [OpenID Connect Discovery](https://swagger.io/docs/specification/authentication/oauth2) section of OpenAPI 3.
