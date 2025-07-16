1. Fix N+1 Query in GetUserDataWithShops()
  - Issue: api/service/shops_service_impl.go:160-172 calls GetShopMemberCount() and
   GetShopVehicleCount() for each shop
  - Fix: Replace with single JOIN query to get all stats at once

  2. Add Critical Composite Indexes
  CREATE INDEX idx_shop_members_user_shop ON shop_members (user_id, shop_id);
  CREATE INDEX idx_shop_vehicles_shop_created ON shop_vehicle (shop_id, save_time);
  CREATE INDEX idx_notifications_vehicle_type ON shop_vehicle_notifications
  (vehicle_id, type);

  3. Implement Query Caching
  - Cache IsUserShopAdmin() and IsUserMemberOfShop() results for 5-10 minutes
  - These are called frequently across the service layer

  4. Add Pagination Support
  - Messages: GetShopMessages() loads all messages without limits
  - Notifications: GetVehicleNotifications() needs cursor-based pagination
  - Members: Large shops could have performance issues

  Priority 2: Critical Race Condition Fixes (High Impact, Medium Risk)

  Transaction Safety Issues

  1. CreateShop + AddMemberToShop Race Condition
  - Location: api/service/shops_service_impl.go:40-51
  - Issue: Two separate database operations without transaction
  - Fix: Wrap in database transaction

  2. LeaveShop Race Condition
  - Location: api/service/shops_service_impl.go:252-263
  - Issue: Check member count, then potentially delete shop - race condition if
  multiple users leave
  - Fix: Use SELECT FOR UPDATE when checking member count

  3. Vehicle Update Conflicts
  - Issue: Concurrent mileage/hours updates could overwrite each other
  - Fix: Add version field for optimistic locking

  4. Invite Code Double-Usage
  - Issue: Multiple users could join with same code simultaneously
  - Fix: Implement distributed locks or use database constraints

  Priority 3: Client Sync Optimizations (Medium Impact, Low Risk)

  API Improvements

  1. Delta Sync Endpoints
  - Add ?since=timestamp parameter for incremental updates
  - Particularly useful for messages and notifications

  2. Lightweight Shop List Endpoint
  - Current GetUserDataWithShops() loads heavy statistics every time
  - Create separate endpoint for basic shop list without stats

  3. Bulk Data Operations
  - Combined endpoints for related data (shop + members + vehicles)
  - Reduce round trips for mobile clients

  4. WebSocket Integration
  - Real-time updates for messages and notifications
  - Batch non-critical updates via REST

  Client-Side Optimizations

  1. Permission Caching
  - Cache user permissions and shop membership on client
  - Invalidate on membership changes

  2. Selective Data Loading
  - Shop statistics only when viewing shop details
  - Lazy load vehicle notifications and items

  Priority 4: Enhanced Error Handling (Medium Impact, Low Risk)

  Input Validation

  1. NIIN Format Validation
  - Military item numbers should follow specific format
  - Add regex validation in request layer

  2. Rate Limiting
  - Implement per-user rate limits on message/notification creation
  - Prevent spam and abuse

  3. Data Constraints
  - Vehicle mileage/hours: min 0, max reasonable values
  - Message length: reasonable character limits
  - Notification types: strict validation for M1, PM, MW

  4. Authorization Edge Cases
  - Cross-shop access validation
  - Stale permission handling
  - Admin privilege escalation prevention

  Implementation Sequence

  Phase 1: Database Performance (Week 1)
  ├── Add composite indexes
  ├── Fix N+1 queries
  └── Implement query caching

  Phase 2: Transaction Safety (Week 2)
  ├── Wrap multi-step operations in transactions
  ├── Fix race conditions with SELECT FOR UPDATE
  └── Add optimistic locking for vehicles

  Phase 3: Client Sync (Week 3)
  ├── Add delta sync endpoints
  ├── Create lightweight endpoints
  └── Implement bulk operations

  Phase 4: Enhanced Validation (Week 4)
  ├── Add input validation
  ├── Implement rate limiting
  └── Strengthen authorization checks

  Specific Code Issues Found

  1. Raw SQL Query Risk (api/repository/shops_repository_impl.go:232-244)
  - Uses raw SQL for member queries - should use JET ORM for consistency

  2. Missing Transaction Boundaries
  - Multiple database operations without proper transaction wrapping
  - Could lead to inconsistent state

  3. Inefficient Statistics Loading
  - Each shop loads member/vehicle counts separately
  - Should be done in bulk with JOINs

  4. No Pagination Strategy
  - Messages, notifications, and large lists could cause memory issues
  - Needs cursor-based pagination for performance