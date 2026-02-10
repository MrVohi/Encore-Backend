create extension if not exists pgcrypto;

-- USERS
create table users (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  username text not null unique,
  mail text not null unique,
  password_hash text, -- nullable for OAuth
  role text not null default 'user',
  created_at timestamptz not null default now()
);

-- OAUTH IDENTITY
create table oauth_identities (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  provider text not null,
  provider_user_id text not null,
  created_at timestamptz not null default now(),
  unique (provider, provider_user_id) 
);

-- TOKEN
create table tokens (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  type text not null,
  token_hash text not null,
  expires_at timestamptz not null,
  created_at timestamptz not null default now()
);

-- ARTIST
create table artists (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  genre text not null,
  image_url text not null,
  preview_url text not null,
  created_at timestamptz not null default now()
);

-- CONCERT
create table concerts (
  id uuid primary key default gen_random_uuid(),
  artist_id uuid not null references artists(id) on delete cascade,
  "when" timestamptz not null,
  country text not null,
  city text not null,
  lat double precision,
  lng double precision,
  capacity integer not null check (capacity >= 0),
  status text not null,
  external_id text not null
);

-- SUIVRE (association Users <-> Artist)
create table follow (
  user_id uuid not null references users(id) on delete cascade,
  artist_id uuid not null references artists(id) on delete cascade,
  primary key (user_id, artist_id)
);

-- ORDERS ("order" is reserved, so use orders)
create table orders (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  status text not null,
  total_amount numeric(10,2) not null check (total_amount >= 0), 
  currency text not null,
  created_at timestamptz not null default now()
);

-- PAYMENTS
create table payments (
  id uuid primary key default gen_random_uuid(),
  order_id uuid not null references orders(id) on delete cascade,
  provider text not null,
  stripe_checkout_session_id text, -- nullable
  stripe_payment_intent_id text,   -- nullable
  status text not null,
  amount numeric(10,2) not null check (amount >= 0),   
  currency text not null,
  created_at timestamptz not null default now()
);

-- TICKET TYPE
create table ticket_types (
  id uuid primary key default gen_random_uuid(),
  concert_id uuid not null references concerts(id) on delete cascade,
  name text not null,
  price numeric(10,2) not null check (price >= 0),
  currency text not null,
  quantity integer not null check (quantity >= 0),
  starts timestamptz not null,
  ends timestamptz not null
);

-- TICKETS
create table tickets (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  concert_id uuid not null references concerts(id) on delete cascade,
  ticket_type_id uuid not null references ticket_types(id) on delete cascade,
  seat text not null,
  status text not null,
  issued_at timestamptz not null,
  used_at timestamptz -- nullable
);
