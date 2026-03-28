# REST Integration

Calling external APIs from microflows -- GET, POST, authentication, and error handling.

## Simple GET

```sql
CREATE MICROFLOW Integration.FetchWebpage ()
RETURNS String AS $Content
BEGIN
  $Content = REST CALL GET 'https://example.com/api/status'
    HEADER Accept = 'application/json'
    TIMEOUT 30
    RETURNS String;
  RETURN $Content;
END;
/
```

## GET with URL Parameters

```sql
CREATE MICROFLOW Integration.SearchProducts (
  $Query: String,
  $Page: Integer
)
RETURNS String AS $Response
BEGIN
  $Response = REST CALL GET 'https://api.example.com/search?q={1}&page={2}' WITH (
    {1} = urlEncode($Query),
    {2} = toString($Page)
  )
    HEADER Accept = 'application/json'
    TIMEOUT 60
    RETURNS String;
  RETURN $Response;
END;
/
```

## POST with JSON Body

```sql
CREATE MICROFLOW Integration.CreateCustomer (
  $Name: String,
  $Email: String
)
RETURNS String AS $Response
BEGIN
  $Response = REST CALL POST 'https://api.example.com/customers'
    HEADER 'Content-Type' = 'application/json'
    BODY '{{"name": "{1}", "email": "{2}"}' WITH (
      {1} = $Name,
      {2} = $Email
    )
    TIMEOUT 30
    RETURNS String;
  RETURN $Response;
END;
/
```

## Basic Authentication

```sql
CREATE MICROFLOW Integration.FetchSecureData (
  $Username: String,
  $Password: String
)
RETURNS String AS $Response
BEGIN
  $Response = REST CALL GET 'https://api.example.com/secure/data'
    HEADER Accept = 'application/json'
    AUTH BASIC $Username PASSWORD $Password
    TIMEOUT 30
    RETURNS String;
  RETURN $Response;
END;
/
```

## Error Handling

Use `ON ERROR WITHOUT ROLLBACK` to catch failures and return a fallback instead of rolling back the transaction:

```sql
CREATE MICROFLOW Integration.SafeAPICall (
  $Url: String
)
RETURNS Boolean AS $Success
BEGIN
  DECLARE $Success Boolean = false;

  $Response = REST CALL GET $Url
    HEADER Accept = 'application/json'
    TIMEOUT 30
    RETURNS String
    ON ERROR WITHOUT ROLLBACK {
      LOG ERROR NODE 'Integration' 'API call failed: ' + $Url;
      RETURN false;
    };

  SET $Success = true;
  RETURN $Success;
END;
/
```
