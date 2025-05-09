# API / Users & Roles

| Method | Endpoint                                                | Description                      |
|:-------|:--------------------------------------------------------|:---------------------------------|
| GET    | [/api/users](#get-apiusers)                             | Retrieve all users.              |
| GET    | [/api/users/{user_id}](#get-apiusersuser_id)            | Retrieve a specific user.        |
| POST   | [/api/users](#post-apiusers)                            | Create a new user.               |
| POST   | [/api/users/api](#post-apiusersapi)                     | Create a new API user.           |
| PUT    | [/api/users/{user_id}](#put-apiusersuser_id)            | Update a user.                   |
| PUT    | [/api/users/{user_id}/role](#put-apiusersuser_idrole)   | Assign a role to a user.         |
| DELETE | [/api/users/{user_id}](#delete-apiusersuser_id)         | Delete a user.                   |
| GET    | [/api/roles/users](#get-apirolesusers)                  | Retrieve all user roles.         |
| GET    | [/api/roles/lists](#get-apiroleslists)                  | Retrieve all list roles.         |
| POST   | [/api/roles/users](#post-apirolesusers)                 | Create a new user role.          |
| POST   | [/api/roles/lists](#post-apiroleslists)                 | Create a new list role.          |
| POST   | [/api/roles/lists/assign](#post-apiroleslistsassign)    | Assign lists to a role.          |
| PUT    | [/api/roles/users/{role_id}](#put-apirolesusersrole_id) | Update a user role.              |
| PUT    | [/api/roles/lists/{role_id}](#put-apiroleslistsrole_id) | Update a list role.              |
| DELETE | [/api/roles/{role_id}](#delete-apirolesrole_id)         | Delete a role.                   |

## GET /api/users

Retrieves all users.

### Parameters
None

### Example response
```json
{
  "data": [
    {
      "id": 1,
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z",
      "username": "admin",
      "email": "admin@example.com",
      "name": "Administrator",
      "type": "user",
      "status": "enabled",
      "avatar": null,
      "loggedin_at": "2023-01-01T00:00:00Z",
      "user_role_id": 1,
      "list_role_id": null,
      "user_role": {
        "id": 1,
        "name": "Super Admin",
        "permissions": ["campaigns:get", "campaigns:manage", "subscribers:get", "subscribers:manage"]
      },
      "list_role": null
    }
  ]
}
```

## POST /api/users/api

Creates a new API user.

### Parameters
| Name        | Type   | Description                                |
|:------------|:-------|:-------------------------------------------|
| username    | string | Username for the API user (required)       |
| name        | string | Display name for the API user              |
| user_role_id| int    | ID of the user role to assign              |
| list_role_id| int    | ID of the list role to assign (optional)   |
| status      | string | Status of the user ("enabled", "disabled") |

### Example request
```json
{
  "username": "api_user",
  "name": "API User",
  "user_role_id": 2,
  "status": "enabled"
}
```

### Example response
```json
{
  "data": {
    "id": 3,
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z",
    "username": "api_user",
    "password": "generated-api-token-shown-only-once",
    "email": "api_user@api",
    "name": "API User",
    "type": "api",
    "status": "enabled",
    "avatar": null,
    "loggedin_at": null,
    "user_role_id": 2,
    "list_role_id": null,
    "user_role": {
      "id": 2,
      "name": "API Role",
      "permissions": ["subscribers:get", "subscribers:manage"]
    },
    "list_role": null
  }
}
```

## PUT /api/users/{user_id}/role

Assigns a role to a user.

### Parameters
| Name        | Type   | Description                                |
|:------------|:-------|:-------------------------------------------|
| user_role_id| int    | ID of the user role to assign              |
| list_role_id| int    | ID of the list role to assign (optional)   |

### Example request
```json
{
  "user_role_id": 2,
  "list_role_id": 1
}
```

### Example response
```json
{
  "data": {
    "id": 3,
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z",
    "username": "user",
    "email": "user@example.com",
    "name": "User",
    "type": "user",
    "status": "enabled",
    "avatar": null,
    "loggedin_at": null,
    "user_role_id": 2,
    "list_role_id": 1,
    "user_role": {
      "id": 2,
      "name": "Editor",
      "permissions": ["subscribers:get", "subscribers:manage"]
    },
    "list_role": {
      "id": 1,
      "name": "List Manager",
      "lists": [
        {
          "id": 1,
          "name": "Newsletter",
          "permissions": ["list:get", "list:manage"]
        }
      ]
    }
  }
}
```

## POST /api/roles/lists/assign

Assigns lists to a role.

### Parameters
| Name        | Type   | Description                                |
|:------------|:-------|:-------------------------------------------|
| role_id     | int    | ID of the role to assign lists to          |
| lists       | array  | Array of list permissions                  |

### Example request
```json
{
  "role_id": 2,
  "lists": [
    {
      "id": 1,
      "permissions": ["list:get", "list:manage"]
    },
    {
      "id": 2,
      "permissions": ["list:get"]
    }
  ]
}
```

### Example response
```json
{
  "data": {
    "id": 2,
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z",
    "name": "List Manager",
    "lists": [
      {
        "id": 1,
        "name": "Newsletter",
        "permissions": ["list:get", "list:manage"]
      },
      {
        "id": 2,
        "name": "Announcements",
        "permissions": ["list:get"]
      }
    ]
  }
}
```
