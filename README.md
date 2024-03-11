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

## NDC REST schema extension

The NDC REST schema extension add `request` information into `functions` and `procedures` so the connector can have more context to initiate HTTP requests to the remote REST service.

```yaml
- request:
    url: /pets/{petId}
    method: get
    type: rest
    headers:
      Authorization: Bearer xxx
    timeout: 30 # seconds, default 30s
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

For procedures, the `body` argument is always treated as the request body. If there is a parameter which has the same name, the tool will rename it to `paramBody`.

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
      body:
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
