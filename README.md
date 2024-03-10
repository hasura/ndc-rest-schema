# NDC Schema Tool

This module includes libraries and tools to convert other API schemas to Native Data Connector (NDC) schema, as well as extend the NDC spec with REST request information.

## Features

- Convert OpenAPI [3.0](https://swagger.io/specification/v3)/[3.1](https://swagger.io/specification/) documentation to NDC schema.

## Installation

## Build from source

**Prerequisites**

- Go 1.21+

**Install**

```go
go install github.com/hasura/ndc-schema-tool
```

## Quick start

Convert an OpenAPI v3 file to NDC schema with the `convert` command. The tool can accept either file path or URL. The output format can be in JSON or YAML, depending on the file extension:

```sh
ndc-schema-tool convert -f https://raw.githubusercontent.com/OAI/OpenAPI-Specification/main/examples/v3.0/petstore.yaml -o petstore.json
```

The output schema can extend from NDC schema with REST information that will be used for NDC REST connector. You can enable the extension with `--rest` flag.

## NDC REST schema extension

The NDC REST schema extension add `request` information into `functions` and `procedures` so the connector can have more context to initiate HTTP requests to the remote REST service.

```yaml
- request:
    url: /pets/{petId}
    method: get
    type: rest
    headers:
      Authorization: Bearer xxx
    timeout: 30
    parameters:
      - name: petId
        in: path
        required: true
        schema:
          type: string
```

The URL can be a relative path or absolute URL. If the URL the relative, there must be a base URL in `settings`:

```yaml
settings:
  url: http://petstore.swagger.io/v1
```

`parameters` include the list of URL and query parameters so the connector can replace values from request arguments.

For procedures, the `data` argument is always treated as the request body. If there is a parameter which has the same name, the tool will rename it to `paramData`.

Full example:

```yaml
settings:
  url: http://petstore.swagger.io/v1
collections: []
functions:
  - request:
      url: /pets/{petId}
      method: get
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: string
    arguments:
      petId:
        description: The id of the pet to retrieve
        type:
          name: String
          type: named
    description: Info for a specific pet
    name: showPetById
    result_type:
      name: Pet
      type: named
procedures:
  - request:
      url: /pets
      method: post
      headers:
        Content-Type: application/json
    arguments:
      data:
        description: Request body of /pets
        type:
          name: Pet
          type: named
    description: Create a pet
    name: createPets
    result_type:
      type: nullable
      underlying_type:
        name: Boolean
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
