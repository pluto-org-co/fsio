# drive2s3

Copy the contents of the Google Drive into an S3 server. It operates with a service account created from the Cloud Console.

```bash
go build ./cmd/drive2s3
```

## Required permissions

This are the required permissions in order to allow access to the service account.

```
https://www.googleapis.com/auth/admin.directory.user.readonly
https://www.googleapis.com/auth/admin.directory.domain.readonly
https://www.googleapis.com/auth/drive
https://www.googleapis.com/auth/gmail.readonly
```

This permissions should be in hand with enabling the following APIs from the Cloud Console.

- Admin SDK API
- Drive API
- Gmail API

## Configuration example

```yaml
workers: 100
interval: 24h
drive:
  account-file: /path/to/redacted/svc-account.json
  subject: "[REDACTED_ADMIN_EMAIL]"
  current-account: true
  shared-drive: true
  other-users: true
s3:
  bucket: bucket-name
  client-id: "[REDACTED_CLIENT_ID]"
  client-secret: "[REDACTED_CLIENT_SECRET]"
  endpoint: "[REDACTED_PRIVATE_ENDPOINT]"
  cache-expiry: 1m
```
