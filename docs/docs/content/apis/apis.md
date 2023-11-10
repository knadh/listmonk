# APIs

All features that are available on the listmonk dashboard are also available as REST-like HTTP APIs that can be interacted with directly. Request and response bodies are JSON. This allows easy scripting of listmonk and integration with other systems, for instance, synchronisation with external subscriber databases.

API requests require BasicAuth authentication with the admin credentials.

> The API section is a work in progress. There may be API calls that are yet to be documented. Please consider contributing to docs.

## OpenAPI (Swagger) spec

The auto-generated OpenAPI (Swagger) specification site for the APIs are available at [**listmonk.app/docs/swagger**](https://listmonk.app/docs/swagger/)

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

| Code  |                                                                          |
| ----- | ------------------------------------------------------------------------ |
|  400  | Missing or bad request parameters or values                              |
|  403  | Session expired or invalidate. Must relogin                              |
|  404  | Request resource was not found                                           |
|  405  | Request method (GET, POST etc.) is not allowed on the requested endpoint |
|  410  | The requested resource is gone permanently                               |
|  429  | Too many requests to the API (rate limiting)                             |
|  500  | Something unexpected went wrong                                          |
|  502  | The backend OMS is down and the API is unable to communicate with it     |
|  503  | Service unavailable; the API is down                                     |
|  504  | Gateway timeout; the API is unreachable                                  |
