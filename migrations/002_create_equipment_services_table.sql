-- Equipment Services Feature Database Schema
-- Migration: 002_create_equipment_services_table.sql

-- Equipment Services table
CREATE TABLE equipment_services (
    id VARCHAR(36) PRIMARY KEY,                    -- UUID, consistent with existing pattern
    shop_id VARCHAR(36) NOT NULL,                  -- Foreign key to shops.id
    equipment_id VARCHAR(36) NOT NULL,             -- Foreign key to shop_vehicle.id
    list_id VARCHAR(36) NOT NULL,                  -- Foreign key to shop_lists.id
    description TEXT NOT NULL,                     -- Service description (1-500 chars)
    service_type TEXT NOT NULL,                    -- Service type (free-form text)
    created_by VARCHAR(255) NOT NULL,              -- User ID who created service (matching users.uid)
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,  -- Track completed vs scheduled services
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    service_date TIMESTAMP NULL,                   -- Scheduled/completed service date
    service_hours INTEGER NULL,                    -- Equipment hours/mileage at service (non-negative)
    
    -- Foreign key constraints with CASCADE DELETE (following existing pattern)
    CONSTRAINT fk_equipment_services_shop 
        FOREIGN KEY (shop_id) REFERENCES shops(id) ON DELETE CASCADE,
    CONSTRAINT fk_equipment_services_equipment 
        FOREIGN KEY (equipment_id) REFERENCES shop_vehicle(id) ON DELETE CASCADE,
    CONSTRAINT fk_equipment_services_list 
        FOREIGN KEY (list_id) REFERENCES shop_lists(id) ON DELETE CASCADE,
    CONSTRAINT fk_equipment_services_user 
        FOREIGN KEY (created_by) REFERENCES users(uid) ON DELETE CASCADE,
    
    -- Validation constraints
    CONSTRAINT chk_service_hours_non_negative 
        CHECK (service_hours IS NULL OR service_hours >= 0),
    CONSTRAINT chk_description_length 
        CHECK (length(description) >= 1 AND length(description) <= 500)
);

-- Indexes for performance (following existing patterns)
CREATE INDEX idx_equipment_services_shop_id ON equipment_services(shop_id);
CREATE INDEX idx_equipment_services_equipment_id ON equipment_services(equipment_id);
CREATE INDEX idx_equipment_services_service_date ON equipment_services(service_date);
CREATE INDEX idx_equipment_services_created_by ON equipment_services(created_by);
CREATE INDEX idx_equipment_services_service_type ON equipment_services(service_type);
CREATE INDEX idx_equipment_services_is_completed ON equipment_services(is_completed);

-- Composite indexes for common query patterns
CREATE INDEX idx_equipment_services_shop_date ON equipment_services(shop_id, service_date);
CREATE INDEX idx_equipment_services_equipment_date ON equipment_services(equipment_id, service_date);
CREATE INDEX idx_equipment_services_shop_completed ON equipment_services(shop_id, is_completed);