# DynamoDB Table Design

## Overview

Este proyecto usa **Single Table Design** simplificado con 2 tablas: `notifications` y `users`.

---

## Table 1: `notifications-{env}`

### Primary Key Design

```
PK (Partition Key): USER#<userID>
SK (Sort Key):      NOTIF#<ISO8601_timestamp>#<ulid>
```

**Ejemplo:**
```
PK: USER#usr_01HQ8X9Y5KNZ4T2B6R
SK: NOTIF#2024-11-02T15:30:00Z#01HQ8XA2B3C4D5E6F7G8H9
```

### Attributes

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `PK` | String | Partition Key | `USER#usr_123` |
| `SK` | String | Sort Key | `NOTIF#2024-11-02T15:30:00Z#abc123` |
| `GSI1PK` | String | GSI Partition Key | `NOTIF#01HQ8XA2B3C4D5E6F7G8H9` |
| `GSI1SK` | String | GSI Sort Key | `NOTIF#01HQ8XA2B3C4D5E6F7G8H9` |
| `id` | String | Unique notification ID (ULID) | `01HQ8XA2B3C4D5E6F7G8H9` |
| `user_id` | String | User ID | `usr_123` |
| `title` | String | Notification title | `"New message"` |
| `content` | String | Notification body | `"You have a new message"` |
| `channel_name` | String | Channel type | `"email"`, `"sms"`, `"push"` |
| `created_at` | String (ISO8601) | Creation timestamp | `2024-11-02T15:30:00Z` |
| `updated_at` | String (ISO8601) | Last update | `2024-11-02T16:00:00Z` |

### GSI1: Query by Notification ID

```
GSI1PK: NOTIF#<id>
GSI1SK: NOTIF#<id>
```

**Purpose:** Direct access to a notification by ID without knowing the user.

**Query:**
```go
// Get notification by ID
dynamodb.Query({
  IndexName: "GSI1",
  KeyConditionExpression: "GSI1PK = :pk",
  ExpressionAttributeValues: {
    ":pk": "NOTIF#abc123"
  }
})
```

### Access Patterns

| Pattern | Key | Example |
|---------|-----|---------|
| List user notifications | `Query(PK=USER#123)` | Get all notifications for user 123 |
| Get notification by ID | `Query(GSI1PK=NOTIF#abc)` | Get specific notification |
| Create notification | `PutItem(PK=USER#123, SK=NOTIF#...)` | Insert new notification |
| Update notification | `UpdateItem(PK=USER#123, SK=NOTIF#...)` | Update existing |
| Delete notification | `DeleteItem(PK=USER#123, SK=NOTIF#...)` | Soft delete (set deleted_at) |

---

## Table 2: `users-{env}`

### Primary Key Design

```
PK (Partition Key): USER#<userID>
SK (Sort Key):      METADATA
```

**Ejemplo:**
```
PK: USER#usr_01HQ8X9Y5KNZ4T2B6R
SK: METADATA
```

### Attributes

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `PK` | String | Partition Key | `USER#usr_123` |
| `SK` | String | Sort Key (always "METADATA") | `METADATA` |
| `id` | String | User ID (ULID) | `usr_123` |
| `email` | String | User email (unique) | `user@example.com` |
| `password_hash` | String | Bcrypt hash | `$2a$10...` |
| `created_at` | String (ISO8601) | Creation timestamp | `2024-11-02T15:30:00Z` |

### GSI1: Query by Email (for login)

```
GSI1PK: EMAIL#<email>
GSI1SK: USER#<id>
```

**Purpose:** Login by email without knowing user ID.

**Query:**
```go
// Find user by email
dynamodb.Query({
  IndexName: "GSI1",
  KeyConditionExpression: "GSI1PK = :email",
  ExpressionAttributeValues: {
    ":email": "EMAIL#user@example.com"
  }
})
```

### Access Patterns

| Pattern | Key | Example |
|---------|-----|---------|
| Get user by ID | `GetItem(PK=USER#123, SK=METADATA)` | User profile |
| Find user by email | `Query(GSI1PK=EMAIL#user@...)` | Login |
| Create user | `PutItem(PK=USER#123, SK=METADATA)` | Signup |
| Update user | `UpdateItem(PK=USER#123, SK=METADATA)` | Update profile |

---

## Why This Design?

### ✅ Benefits

1. **Efficient queries**: List user notifications without scanning entire table
2. **Natural sorting**: SK with timestamp sorts chronologically
3. **Direct access**: GSI allows ID-based lookups
4. **Scalability**: DynamoDB distributes data across partitions by PK

### ⚠️ Trade-offs

1. **No joins**: Can't query "notifications with user email" in one request
2. **Denormalization**: If user changes email, notifications keep old user_id
3. **GSI cost**: Extra writes (2x write units) for maintaining GSI

---

## Cost Estimation (Free Tier)

DynamoDB Free Tier (Permanent):
- **25 GB storage**
- **25 WCU** (Write Capacity Units) = 25 writes/second
- **25 RCU** (Read Capacity Units) = 25 reads/second

**Our usage:**
- 1 notification = ~1KB = 1 WCU
- 1 read = 1 RCU
- GSI writes = 2x (main table + GSI)

**Example:**
- 1000 notifications/day = ~0.01 WCU average (well within free tier)
- 10,000 reads/day = ~0.1 RCU average

