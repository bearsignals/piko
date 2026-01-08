# Milestone 7: Shared Services

**Priority:** Low
**Depends on:** M1-M2 (Core + Lifecycle)
**Unlocks:** Resource efficiency for common services

## Goal

Some services (redis, elasticsearch, jaeger) can be shared across environments instead of running multiple instances. This saves resources and simplifies debugging.

## Success Criteria

```bash
$ piko share redis
✓ Marked redis as shared
✓ Created shared network: piko-myapp-shared
✓ Started shared redis container

$ piko create feature-auth
# redis is NOT started in this environment
# Instead, linked to shared redis via network

$ piko create feature-pay
# Also linked to shared redis

$ piko isolate redis
✓ Marked redis as isolated
# Future environments will have their own redis
```

## Tasks

### 6.1 Shared Services Table
- [ ] Add to SQLite schema:
  ```sql
  CREATE TABLE shared_services (
      id INTEGER PRIMARY KEY,
      project_id INTEGER REFERENCES project(id),
      service_name TEXT NOT NULL,
      container_name TEXT,
      network TEXT NOT NULL,
      UNIQUE(project_id, service_name)
  );
  ```

### 6.2 Share Command
- [ ] `piko share <service>`
- [ ] Validate service exists in compose file
- [ ] Create shared network: `piko-<project>-shared`
- [ ] Start service in shared network:
  ```bash
  docker compose -p piko-<project>-shared \
    -f docker-compose.yml \
    up -d <service>
  ```
- [ ] Record in shared_services table

### 6.3 Isolate Command
- [ ] `piko isolate <service>`
- [ ] Remove from shared_services table
- [ ] Stop shared container if no consumers
- [ ] Future environments will run their own

### 6.4 Override Generation Update
- [ ] Check shared_services table
- [ ] For shared services:
  - Exclude from `services:` section in override
  - Add `piko-<project>-shared` to networks
- [ ] Generate network links in override:
  ```yaml
  networks:
    default:
      name: piko-myapp-feature-auth
    piko-shared:
      external: true
      name: piko-myapp-shared
  ```

### 6.5 Compose Override for Shared
- [ ] Non-shared services connect to both networks
- [ ] Shared service only in shared network
- [ ] Service discovery via container name

### 6.6 Lifecycle Integration
- [ ] `piko up`: Don't start shared services
- [ ] `piko down`: Don't stop shared services
- [ ] `piko destroy`: Unlink from shared network
- [ ] Track consumers of shared services

### 6.7 Shared Service Cleanup
- [ ] When last environment destroyed, stop shared service
- [ ] Or: `piko down --shared` to stop shared services
- [ ] Or: Leave running (cheap, convenient)

## Networking Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│  piko-myapp-shared network                                      │
│  ┌──────────┐                                                   │
│  │  redis   │  ← shared instance                                │
│  └────┬─────┘                                                   │
└───────┼─────────────────────────────────────────────────────────┘
        │
        ├──────────────────────┐
        │                      │
┌───────▼─────────────────┐  ┌─▼───────────────────────┐
│ piko-myapp-feature-auth │  │ piko-myapp-feature-pay  │
│ ┌─────┐ ┌────┐          │  │ ┌─────┐ ┌────┐          │
│ │ app │ │ db │          │  │ │ app │ │ db │          │
│ └─────┘ └────┘          │  │ └─────┘ └────┘          │
└─────────────────────────┘  └─────────────────────────┘
```

## Test Cases

1. **Share service**: Creates shared container
2. **Create with shared**: Doesn't duplicate service
3. **Multiple envs**: All connect to same shared
4. **Isolate service**: Stops sharing
5. **Create after isolate**: Has own instance
6. **Destroy all envs**: Shared still runs (optional cleanup)

## Definition of Done

- [ ] `piko share <service>` creates shared instance
- [ ] `piko isolate <service>` stops sharing
- [ ] New environments link to shared services
- [ ] Shared services not duplicated per environment
- [ ] Network connectivity works between envs and shared
- [ ] Cleanup works correctly
