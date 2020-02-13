# API Tests Users and Configuration
---
It is necessary for the various api tests to run as a user locally and in INT/STAGE/PROD. 
To enable testing both when a request is authorized and when a request is unauthorized, 
it is necessary to have at least two users pre-configured with the proper authorization 
in each environment.

## User list

The following users are used by api-tests:

| Environment | ID | Config Key | Authorization description |
|---|---|---|---|
| INT | TODO | superadmin | Has access to everything, authoization should succeed across all requests |
| INT | TODO | noauth | Has access to nothing. Is not "banned", but has no role bindings.
| STAGE | TODO | superadmin | Has access to everything, authoization should succeed across all requests |
| STAGE | TODO | noauth | Has access to nothing. Is not "banned", but has no role bindings.
| PROD | TODO | superadmin | Has access to everything, authoization should succeed across all requests |
| PROD | TODO | noauth | Has access to nothing. Is not "banned", but has no role bindings.

### User secrets

Each user created will need the corresponding credentials supplied to the test run. The
integration credentials should be safe to keep in the code repositories. However, the
credentials for stage/prod should be kept in vault.

Below is a table of the proposed secret keys and values for each environment:

| Environment | Key | Value |
| ---| --- | --- |
| INT | superadmin.token | TODO |
| INT | noauth.token | TODO |
| STAGE | superadmin.token | Stored in vault |
| STAGE | noauth.token | Stored in vault |
| PROD | superadmin.token | Stored in vault |
| PROD | noauth.token | Stored in vault |

## SDK Connections

Today (as of c0c933af) only one connection to the SDK is made and is shared across all 
tests that are run. It is necessary to support many connections with different tokens 
corresponding to users of various authorization levels.

We should determine the proper way to establish multiple connections to the SDK without
relying on any set environment variables.

## Test Config

The test config should contain a map of SDK connections that are created when the test suite
is started. The keys of the map correspond to user config keys in the [User List].

Each test will need to select which connection to use from the list.

For example, today a test would pull the sdk connection client like this:
```
client := ocm.Connection.AccountsMgmt().V1()
```

It may need to select from the map, like this:
```
client := ocm.Connections["superadmin"].AccountsMgmt().V1()
```
