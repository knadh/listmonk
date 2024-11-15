# APIs

All features that are available on the listmonk dashboard are also available as REST-like HTTP APIs that can be interacted with directly. Request and response bodies are JSON. This allows easy scripting of listmonk and integration with other systems, for instance, synchronisation with external subscriber databases.

!!! note
    If you come across API calls that are yet to be documented, please consider contributing to docs.


## Auth
HTTP API requests support BasicAuth and a Authorization `token` headers. API users and tokens with the required permissions can be created and managed on the admin UI (Admin -> Users).

##### BasicAuth example
```shell
curl -u "api_user:token" http://localhost:9000/api/lists
```

##### Authorization token example
```shell
curl -H "Authorization: token api_user:token" http://localhost:9000/api/lists
```

## Permissions
**User role**: Permissions allowed for a user are defined as a *User role* (Admin -> User roles) and then attached to a user. 

**List role**: Read / write permissions per-list can be defined as a *List role* (Admin -> User roles) and then attached to a user. 

In a *User role*, `lists:get_all` or `lists:manage_all` permission supercede and override any list specific permissions for a user defined in a *List role*.

To manage lists and subscriber list subscriptions via API requests, ensure that the appropriate permissions are attached to the API user.

______________________________________________________________________

## Response structure

### Successful request

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
    "data": {}
}
```

All responses from the API server are JSON with the content-type application/json unless explicitly stated otherwise. A successful 200 OK response always has a JSON response body with a status key with the value success. The data key contains the full response payload.

### Failed request

```http
HTTP/1.1 500 Server error
Content-Type: application/json

{
    "message": "Error message"
}
```

A failure response is preceded by the corresponding 40x or 50x HTTP header. There may be an optional `data` key with additional payload.

### Timestamps

All timestamp fields are in the format `2019-01-01T09:00:00.000000+05:30`. The seconds component is suffixed by the milliseconds, followed by the `+` and the timezone offset.

### Common HTTP error codes

| Code  |                                                                             |
| ----- | ----------------------------------------------------------------------------|
|  400  | Missing or bad request parameters or values                                 |
|  403  | Session expired or invalidate. Must relogin                                 |
|  404  | Request resource was not found                                              |
|  405  | Request method (GET, POST etc.) is not allowed on the requested endpoint    |
|  410  | The requested resource is gone permanently                                  |
|  422  | Unprocessable entity. Unable to process request as it contains invalid data |
|  429  | Too many requests to the API (rate limiting)                                |
|  500  | Something unexpected went wrong                                             |
|  502  | The backend OMS is down and the API is unable to communicate with it        |
|  503  | Service unavailable; the API is down                                        |
|  504  | Gateway timeout; the API is unreachable                                     |


## OpenAPI (Swagger) spec

The auto-generated OpenAPI (Swagger) specification site for the APIs are available at [**listmonk.app/docs/swagger**](https://listmonk.app/docs/swagger/)

