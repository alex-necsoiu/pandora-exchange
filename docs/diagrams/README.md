# Architecture Diagrams

> **Visual documentation for Pandora Exchange User Service**  
> **Last Updated:** November 8, 2025

---

## Overview

This directory contains Mermaid diagrams documenting the architecture, workflows, and integrations of the Pandora Exchange platform.

**Why Mermaid?**
- ✅ Version controlled (text-based, Git-friendly)
- ✅ Renders in GitHub, GitLab, Markdown viewers
- ✅ Easy to update (no external tools required)
- ✅ Consistent styling
- ✅ No binary files

**Viewing Diagrams:**
- **GitHub/GitLab:** Automatic rendering in README files
- **VS Code:** Install [Markdown Preview Mermaid Support](https://marketplace.visualstudio.com/items?itemName=bierner.markdown-mermaid)
- **IntelliJ/GoLand:** Built-in Mermaid support in Markdown preview
- **Online:** [Mermaid Live Editor](https://mermaid.live/)

---

## Diagram Index

| Diagram | Type | Description | Complexity |
|---------|------|-------------|------------|
| **[User Registration Flow](#user-registration-flow)** | Sequence | User registration process | Simple |
| **[Authentication Flow](#authentication-flow)** | Sequence | Login and token refresh | Medium |
| **[Clean Architecture](#clean-architecture)** | Flowchart | Layer dependencies | Simple |
| **[Event Publishing Flow](#event-publishing-flow)** | Sequence | Event-driven architecture | Medium |

---

## User Registration Flow

**Purpose:** Document the complete user registration process including validation, password hashing, database storage, and event publishing.

```mermaid
sequenceDiagram
    actor User
    participant Client
    participant API as User Service API
    participant Service as User Service
    participant DB as PostgreSQL
    participant Events as Event Publisher
    participant Queue as Message Queue

    User->>Client: Enter registration details
    Client->>API: POST /auth/register<br/>{email, password, first_name, last_name}
    
    API->>API: Validate request<br/>(email format, password strength)
    
    alt Invalid input
        API-->>Client: 400 Bad Request<br/>{error: "Validation failed"}
        Client-->>User: Show validation errors
    end
    
    API->>Service: CreateUser(ctx, RegisterUserDTO)
    
    Service->>Service: Validate business rules<br/>(email uniqueness)
    Service->>Service: Hash password<br/>(Argon2id)
    
    Service->>DB: BEGIN TRANSACTION
    Service->>DB: INSERT INTO users<br/>(email, password_hash, ...)
    
    alt Email already exists
        DB-->>Service: Error: unique constraint violation
        Service->>DB: ROLLBACK
        Service-->>API: ErrUserAlreadyExists
        API-->>Client: 409 Conflict<br/>{error: "USER_ALREADY_EXISTS"}
        Client-->>User: Show error message
    end
    
    DB-->>Service: User created (user_id)
    
    Service->>Events: PublishEvent("user.registered", UserEvent)
    Events->>Queue: Publish to exchange
    
    Service->>DB: COMMIT
    
    Service-->>API: User{id, email, ...}
    API-->>Client: 201 Created<br/>{user: {...}}
    Client-->>User: Registration successful<br/>Redirect to dashboard
    
    Note over Queue: Async consumers:<br/>- Send welcome email<br/>- Create wallet<br/>- Update analytics
```

**Key Flows:**
1. **Happy Path:** User registration succeeds, event published
2. **Validation Error:** Invalid input rejected before database access
3. **Duplicate Email:** Database constraint enforced, transaction rolled back

---

## Authentication Flow

**Purpose:** Document JWT-based authentication including login, token refresh, and token rotation.

```mermaid
sequenceDiagram
    actor User
    participant Client
    participant API as User Service API
    participant Service as User Service
    participant DB as PostgreSQL
    participant Vault as HashiCorp Vault

    %% LOGIN FLOW
    rect rgb(200, 220, 250)
    Note over User,Vault: LOGIN FLOW
    
    User->>Client: Enter credentials
    Client->>API: POST /auth/login<br/>{email, password}
    
    API->>Service: ValidateCredentials(ctx, email, password)
    Service->>DB: SELECT * FROM users<br/>WHERE email = $1
    
    alt User not found
        DB-->>Service: No rows
        Service-->>API: ErrInvalidCredentials
        API-->>Client: 401 Unauthorized
        Client-->>User: Invalid credentials
    end
    
    DB-->>Service: User{id, password_hash, ...}
    Service->>Service: Verify password<br/>(Argon2id compare)
    
    alt Wrong password
        Service-->>API: ErrInvalidCredentials
        API-->>Client: 401 Unauthorized
        Client-->>User: Invalid credentials
    end
    
    Service->>Vault: Get JWT secret
    Vault-->>Service: JWT secret
    
    Service->>Service: Generate Access Token<br/>(15 min expiry)
    Service->>Service: Generate Refresh Token<br/>(7 days expiry)
    
    Service->>DB: INSERT INTO refresh_tokens<br/>(user_id, token_hash, expires_at)
    DB-->>Service: Token stored
    
    Service-->>API: Tokens{access, refresh}
    API-->>Client: 200 OK<br/>{access_token, refresh_token}
    Client->>Client: Store tokens<br/>(memory + HTTPOnly cookie)
    Client-->>User: Login successful
    end
    
    %% TOKEN REFRESH FLOW
    rect rgb(250, 220, 200)
    Note over User,Vault: TOKEN REFRESH FLOW (after 15 min)
    
    Client->>Client: Access token expired
    Client->>API: POST /auth/refresh<br/>{refresh_token}
    
    API->>Service: RefreshTokens(ctx, refreshToken)
    Service->>DB: SELECT * FROM refresh_tokens<br/>WHERE token_hash = $1
    
    alt Invalid/expired refresh token
        DB-->>Service: No rows
        Service-->>API: ErrInvalidToken
        API-->>Client: 401 Unauthorized
        Client-->>User: Session expired<br/>Redirect to login
    end
    
    DB-->>Service: RefreshToken{user_id, expires_at}
    Service->>DB: DELETE FROM refresh_tokens<br/>WHERE token_hash = $1
    
    Service->>Vault: Get JWT secret
    Vault-->>Service: JWT secret
    
    Service->>Service: Generate new Access Token<br/>(15 min expiry)
    Service->>Service: Generate new Refresh Token<br/>(7 days expiry)
    
    Service->>DB: INSERT INTO refresh_tokens<br/>(user_id, new_token_hash, expires_at)
    
    Service-->>API: Tokens{access, refresh}
    API-->>Client: 200 OK<br/>{access_token, refresh_token}
    Client->>Client: Update tokens
    end
```

**Key Flows:**
1. **Login:** Credentials validated, tokens generated and stored
2. **Token Refresh:** Old refresh token revoked, new tokens issued (rotation)
3. **Error Cases:** Invalid credentials, expired tokens, token theft detection

**Security Features:**
- Passwords hashed with Argon2id (never stored plaintext)
- JWT secret retrieved from Vault (never hardcoded)
- Refresh tokens are one-time use (rotation prevents replay attacks)
- Refresh tokens hashed before database storage

---

## Clean Architecture

**Purpose:** Document the layered architecture and dependency rules (Inner layers know nothing about outer layers).

```mermaid
flowchart TD
    subgraph External["🌐 External Layer"]
        Client[Client Applications]
        DB[(PostgreSQL)]
        Vault[HashiCorp Vault]
        Queue[Message Queue]
    end
    
    subgraph Transport["📡 Transport Layer (Outer)"]
        HTTP[HTTP Handlers<br/>Gin Router]
        GRPC[gRPC Handlers<br/>Planned]
        Middleware[Middleware<br/>Auth, Logging, etc.]
    end
    
    subgraph Service["⚙️ Service Layer (Application)"]
        UserService[User Service<br/>Business Logic]
        AuditService[Audit Cleanup Job]
    end
    
    subgraph Domain["💼 Domain Layer (Core)"]
        Models[Domain Models<br/>User, RefreshToken, AuditLog]
        Interfaces[Interfaces<br/>UserRepository, EventPublisher]
        Errors[Domain Errors<br/>ErrUserNotFound, etc.]
    end
    
    subgraph Infra["🔧 Infrastructure Layer (Outer)"]
        UserRepo[User Repository<br/>PostgreSQL impl]
        AuditRepo[Audit Repository<br/>PostgreSQL impl]
        EventPub[Event Publisher<br/>RabbitMQ impl]
        VaultClient[Vault Client]
    end
    
    Client -->|HTTP Requests| HTTP
    Client -->|gRPC Requests| GRPC
    
    HTTP --> Middleware
    GRPC --> Middleware
    Middleware -->|Calls| UserService
    
    UserService -->|Uses| Interfaces
    UserService -->|Returns| Models
    UserService -->|Throws| Errors
    
    AuditService -->|Uses| Interfaces
    
    UserRepo -->|Implements| Interfaces
    AuditRepo -->|Implements| Interfaces
    EventPub -->|Implements| Interfaces
    
    UserRepo -->|Queries| DB
    AuditRepo -->|Queries| DB
    VaultClient -->|Fetches Secrets| Vault
    EventPub -->|Publishes| Queue
    
    style Domain fill:#e1f5ff,stroke:#01579b,stroke-width:3px
    style Service fill:#fff9c4,stroke:#f57f17,stroke-width:2px
    style Transport fill:#f3e5f5,stroke:#4a148c,stroke-width:2px
    style Infra fill:#e8f5e9,stroke:#1b5e20,stroke-width:2px
    style External fill:#fce4ec,stroke:#880e4f,stroke-width:2px
    
    classDef coreStyle fill:#bbdefb,stroke:#0d47a1
    class Models,Interfaces,Errors coreStyle
```

**Dependency Rules:**

```
Domain (Core)
    ↑
    │ depends on
    │
Service Layer
    ↑
    │ depends on
    │
Transport Layer → Infrastructure Layer
    ↑                    ↑
    │                    │
External Systems ←───────┘
```

**Key Principles:**
1. **Domain Layer** (innermost):
   - No external dependencies
   - Pure business logic
   - Defines interfaces (ports)
   - Stable, rarely changes

2. **Service Layer**:
   - Orchestrates domain logic
   - Depends only on domain interfaces
   - Transaction boundaries
   - Use cases implementation

3. **Transport Layer** (HTTP, gRPC):
   - Adapters for external communication
   - Maps DTOs ↔ Domain models
   - Input validation
   - Error mapping

4. **Infrastructure Layer**:
   - Implements domain interfaces
   - Database access (sqlc)
   - Event publishing
   - External service clients

**Benefits:**
- ✅ Testable (mock interfaces easily)
- ✅ Framework-independent core
- ✅ Database-independent core
- ✅ Easy to swap implementations

---

## Event Publishing Flow

**Purpose:** Document event-driven architecture for async operations and cross-service communication.

```mermaid
sequenceDiagram
    participant Service as User Service
    participant Publisher as Event Publisher
    participant Queue as Message Queue<br/>(RabbitMQ)
    participant Email as Email Service
    participant Wallet as Wallet Service
    participant Analytics as Analytics Service
    
    rect rgb(200, 250, 200)
    Note over Service,Analytics: USER REGISTERED EVENT
    
    Service->>Service: User created in database
    Service->>Publisher: PublishEvent("user.registered", event)
    
    Publisher->>Publisher: Serialize event (JSON)
    Publisher->>Queue: Publish to "user.events" exchange
    Queue->>Queue: Route to bound queues
    
    par Async Consumers
        Queue->>Email: Consume event
        Email->>Email: Send welcome email
        Email->>Email: Ack message
    and
        Queue->>Wallet: Consume event
        Wallet->>Wallet: Create user wallet
        Wallet->>Wallet: Ack message
    and
        Queue->>Analytics: Consume event
        Analytics->>Analytics: Track registration
        Analytics->>Analytics: Ack message
    end
    
    Note over Service: Service continues<br/>(not blocked by consumers)
    end
    
    rect rgb(250, 230, 200)
    Note over Service,Analytics: USER DELETED EVENT
    
    Service->>Service: User soft deleted
    Service->>Publisher: PublishEvent("user.deleted", event)
    Publisher->>Queue: Publish to "user.events" exchange
    
    par Cleanup Consumers
        Queue->>Wallet: Consume event
        Wallet->>Wallet: Lock wallet<br/>Withdraw funds
        Wallet->>Wallet: Ack message
    and
        Queue->>Analytics: Consume event
        Analytics->>Analytics: Anonymize user data
        Analytics->>Analytics: Ack message
    end
    end
```

**Event Schema:**

```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "event_type": "user.registered",
  "timestamp": "2025-11-08T10:30:00Z",
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "alice@example.com",
    "first_name": "Alice",
    "last_name": "Smith",
    "role": "user"
  },
  "metadata": {
    "service": "user-service",
    "version": "1.2.3"
  }
}
```

**Event Types:**
- `user.registered` - New user created
- `user.profile.updated` - Profile changed
- `user.kyc.updated` - KYC status changed
- `user.deleted` - User account deleted
- `user.logged_in` - User authenticated

**Benefits:**
- ✅ Loose coupling between services
- ✅ Async processing (non-blocking)
- ✅ Scalability (add consumers independently)
- ✅ Reliability (message queue guarantees delivery)
- ✅ Audit trail (all events logged)

---

## Editing Diagrams

### Mermaid Syntax

**Sequence Diagram:**
```mermaid
sequenceDiagram
    Alice->>Bob: Hello Bob!
    Bob-->>Alice: Hi Alice!
```

**Flowchart:**
```mermaid
flowchart LR
    A[Start] --> B{Decision}
    B -->|Yes| C[Do something]
    B -->|No| D[Do something else]
    C --> E[End]
    D --> E
```

**Learn More:**
- [Mermaid Documentation](https://mermaid.js.org/intro/)
- [Mermaid Live Editor](https://mermaid.live/) (test diagrams)

---

## Contributing

**Adding a new diagram:**

1. **Create .mmd file** (optional, for complex diagrams)
   ```bash
   touch docs/diagrams/new-diagram.mmd
   ```

2. **Write Mermaid syntax**
   ```mermaid
   flowchart TD
       A --> B
   ```

3. **Add to this README**
   - Update diagram index table
   - Add section with diagram + description

4. **Test rendering**
   - Preview in VS Code / IntelliJ
   - Or paste into [Mermaid Live](https://mermaid.live/)

5. **Submit PR**
   - Include screenshot of rendered diagram in PR description

---

## References

- [Mermaid Documentation](https://mermaid.js.org/)
- [Mermaid Live Editor](https://mermaid.live/)
- [ARCHITECTURE.md](../../ARCHITECTURE.md) - System architecture
- [User Service Documentation](../services/user-service.md)

---

**Last Updated:** November 8, 2025  
**Maintained By:** Engineering Team
