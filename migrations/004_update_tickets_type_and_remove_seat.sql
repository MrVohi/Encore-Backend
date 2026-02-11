-- Update tickets: remove seat and store ticket type as string instead of id

alter table tickets
  add column ticket_type text;

update tickets t
set ticket_type = coalesce(tt.name, 'General')
from ticket_types tt
where t.ticket_type_id = tt.id;

alter table tickets
  alter column ticket_type set not null;

alter table tickets
  drop constraint if exists tickets_ticket_type_id_fkey;

alter table tickets
  drop column ticket_type_id;

alter table tickets
  drop column seat;
