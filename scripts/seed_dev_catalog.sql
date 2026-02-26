BEGIN;

INSERT INTO categories (name, slug, parent_id)
VALUES
  ('Outerwear', 'outerwear', NULL),
  ('Apparel', 'apparel', NULL),
  ('Footwear', 'footwear', NULL),
  ('Accessories', 'accessories', NULL),
  ('Training', 'training', NULL),
  ('Lifestyle', 'lifestyle', NULL)
ON CONFLICT ((LOWER(slug))) DO UPDATE
SET name = EXCLUDED.name;

INSERT INTO products (
  name,
  slug,
  description,
  price_cents,
  currency,
  stock,
  is_active,
  sku,
  weight_grams,
  image_url
)
VALUES
  ('Aero Running Jacket', 'aero-running-jacket', 'Lightweight weather-resistant jacket for daily runs and travel.', 12900, 'USD', 24, TRUE, 'OUT-AERO-001', 410, 'https://images.unsplash.com/photo-1521572163474-6864f9cf17ab'),
  ('Northline Down Parka', 'northline-down-parka', 'Insulated parka designed for cold city commutes and outdoor weekends.', 21900, 'USD', 14, TRUE, 'OUT-NORTH-002', 980, 'https://images.unsplash.com/photo-1541099649105-f69ad21f3246'),
  ('Stormshell Packable Windbreaker', 'stormshell-packable-windbreaker', 'Packable shell that blocks wind with breathable comfort.', 9900, 'USD', 31, TRUE, 'OUT-STORM-003', 360, 'https://images.unsplash.com/photo-1483985988355-763728e1935b'),
  ('Flex Everyday Tee', 'flex-everyday-tee', 'Soft stretch tee for all-day wear with moisture-managing fabric.', 3200, 'USD', 80, TRUE, 'APP-FLEX-004', 170, 'https://images.unsplash.com/photo-1521572163474-6864f9cf17ab'),
  ('Core Training Set', 'core-training-set', 'Breathable tee and short combo optimized for gym sessions.', 8900, 'USD', 27, TRUE, 'APP-CORE-005', 530, 'https://images.unsplash.com/photo-1574180566232-aaad1b5b8450'),
  ('Momentum Performance Shorts', 'momentum-performance-shorts', 'Four-way stretch shorts with secure zip pocket and liner.', 5400, 'USD', 43, TRUE, 'APP-MOM-006', 190, 'https://images.unsplash.com/photo-1514996937319-344454492b37'),
  ('Altitude Compression Top', 'altitude-compression-top', 'Supportive compression layer for high-intensity training.', 4700, 'USD', 36, TRUE, 'APP-ALT-007', 160, 'https://images.unsplash.com/photo-1521572267360-ee0c2909d518'),
  ('Cloudstrike Runner', 'cloudstrike-runner', 'Responsive running shoe with cushioned midsole and neutral support.', 13900, 'USD', 22, TRUE, 'FTW-CLOUD-008', 620, 'https://images.unsplash.com/photo-1542291026-7eec264c27ff'),
  ('Terrain Hybrid Trainer', 'terrain-hybrid-trainer', 'Cross-training shoe for treadmills, circuits, and light trail.', 11900, 'USD', 18, TRUE, 'FTW-TERR-009', 670, 'https://images.unsplash.com/photo-1460353581641-37baddab0fa2'),
  ('Velocity Court Sneaker', 'velocity-court-sneaker', 'Minimal everyday sneaker with grip-focused outsole.', 10900, 'USD', 25, TRUE, 'FTW-VELO-010', 590, 'https://images.unsplash.com/photo-1491553895911-0055eca6402d'),
  ('Urban Carry Pack', 'urban-carry-pack', 'Commuter backpack with laptop sleeve and modular compartments.', 7400, 'USD', 39, TRUE, 'ACC-URBN-011', 820, 'https://images.unsplash.com/photo-1491637639811-60e2756cc1c7'),
  ('Summit Weekender Duffel', 'summit-weekender-duffel', 'Durable duffel bag with water-resistant base and shoe pocket.', 8600, 'USD', 20, TRUE, 'ACC-SUMM-012', 900, 'https://images.unsplash.com/photo-1553062407-98eeb64c6a62'),
  ('Pace Hydration Bottle', 'pace-hydration-bottle', 'Vacuum-insulated steel bottle to keep drinks cold for hours.', 2400, 'USD', 95, TRUE, 'ACC-PACE-013', 280, 'https://images.unsplash.com/photo-1523362628745-0c100150b504'),
  ('Transit Tech Cap', 'transit-tech-cap', 'Lightweight cap with perforated panels and sweatband.', 2200, 'USD', 67, TRUE, 'ACC-TRANS-014', 95, 'https://images.unsplash.com/photo-1521369909029-2afed882baee'),
  ('Zen Recovery Hoodie', 'zen-recovery-hoodie', 'Relaxed fit hoodie built for cooldown and travel comfort.', 7800, 'USD', 28, TRUE, 'LIFE-ZEN-015', 620, 'https://images.unsplash.com/photo-1503341504253-dff4815485f1'),
  ('City Motion Jogger', 'city-motion-jogger', 'Tapered joggers with articulated knees and soft interior.', 6900, 'USD', 33, TRUE, 'LIFE-CITY-016', 430, 'https://images.unsplash.com/photo-1473966968600-fa801b869a1a'),
  ('Studio Grip Socks (3-Pack)', 'studio-grip-socks-3-pack', 'Non-slip training socks engineered for stability and comfort.', 1800, 'USD', 120, TRUE, 'TRN-SOCK-017', 140, 'https://images.unsplash.com/photo-1586350977771-b3b0abd50c82'),
  ('Power Loop Resistance Bands', 'power-loop-resistance-bands', 'Five-band resistance set for mobility and strength workouts.', 3500, 'USD', 52, TRUE, 'TRN-LOOP-018', 450, 'https://images.unsplash.com/photo-1517836357463-d25dfeac3438')
ON CONFLICT (sku) DO UPDATE
SET
  name = EXCLUDED.name,
  slug = EXCLUDED.slug,
  description = EXCLUDED.description,
  price_cents = EXCLUDED.price_cents,
  currency = EXCLUDED.currency,
  stock = EXCLUDED.stock,
  is_active = EXCLUDED.is_active,
  weight_grams = EXCLUDED.weight_grams,
  image_url = EXCLUDED.image_url;

WITH product_ref AS (
  SELECT id, slug FROM products
),
category_ref AS (
  SELECT id, slug FROM categories
)
INSERT INTO product_categories (product_id, category_id)
SELECT p.id, c.id
FROM (
  VALUES
    ('aero-running-jacket', 'outerwear'),
    ('aero-running-jacket', 'training'),
    ('northline-down-parka', 'outerwear'),
    ('stormshell-packable-windbreaker', 'outerwear'),
    ('flex-everyday-tee', 'apparel'),
    ('flex-everyday-tee', 'lifestyle'),
    ('core-training-set', 'apparel'),
    ('core-training-set', 'training'),
    ('momentum-performance-shorts', 'apparel'),
    ('momentum-performance-shorts', 'training'),
    ('altitude-compression-top', 'apparel'),
    ('altitude-compression-top', 'training'),
    ('cloudstrike-runner', 'footwear'),
    ('cloudstrike-runner', 'training'),
    ('terrain-hybrid-trainer', 'footwear'),
    ('terrain-hybrid-trainer', 'training'),
    ('velocity-court-sneaker', 'footwear'),
    ('velocity-court-sneaker', 'lifestyle'),
    ('urban-carry-pack', 'accessories'),
    ('urban-carry-pack', 'lifestyle'),
    ('summit-weekender-duffel', 'accessories'),
    ('pace-hydration-bottle', 'accessories'),
    ('pace-hydration-bottle', 'training'),
    ('transit-tech-cap', 'accessories'),
    ('transit-tech-cap', 'lifestyle'),
    ('zen-recovery-hoodie', 'apparel'),
    ('zen-recovery-hoodie', 'lifestyle'),
    ('city-motion-jogger', 'apparel'),
    ('city-motion-jogger', 'lifestyle'),
    ('studio-grip-socks-3-pack', 'accessories'),
    ('studio-grip-socks-3-pack', 'training'),
    ('power-loop-resistance-bands', 'training')
) AS mapping(product_slug, category_slug)
JOIN product_ref p ON p.slug = mapping.product_slug
JOIN category_ref c ON c.slug = mapping.category_slug
ON CONFLICT DO NOTHING;

COMMIT;
