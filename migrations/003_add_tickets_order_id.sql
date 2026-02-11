-- Add order_id to tickets for Stripe checkout linkage
alter table tickets
  add column if not exists order_id uuid references orders(id) on delete cascade;
