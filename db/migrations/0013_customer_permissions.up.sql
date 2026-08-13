-- 0013 — the permissions a CUSTOMER holds.
--
-- 0011 granted permissions to the five staff roles and, by omission, gave the
-- customer role none at all. Deny-by-default then means a signed-in customer
-- can do nothing — which is the correct failure mode, and exactly why the gap
-- showed up the moment /me returned an empty permission list rather than
-- surfacing later as a mysterious 403 at checkout.
--
-- These are all OWN-scoped. Ownership is still checked per row in the
-- repository: holding order.view.own says "you may view your own orders", it
-- does not say which orders are yours. IDOR is the top risk here (PROMPT §14),
-- so the permission and the ownership check are two separate controls.

INSERT INTO permission (id, code, description) VALUES
 ('00000000-0000-7000-8000-000000000131','profile.manage','Edit your own profile'),
 ('00000000-0000-7000-8000-000000000132','address.manage','Manage your own delivery addresses'),
 ('00000000-0000-7000-8000-000000000133','order.create','Place an order'),
 ('00000000-0000-7000-8000-000000000134','order.view.own','View your own orders'),
 ('00000000-0000-7000-8000-000000000135','order.cancel.own','Cancel your own unpaid order before the cut-off'),
 ('00000000-0000-7000-8000-000000000136','payment.proof.upload','Upload proof of transfer for your own order'),
 ('00000000-0000-7000-8000-000000000137','package.view.own','View your own packages and credit balance'),
 ('00000000-0000-7000-8000-000000000138','delivery.view.own','View your own deliveries'),
 ('00000000-0000-7000-8000-000000000139','delivery.schedule.own','Pick, skip or reschedule your own package deliveries');

INSERT INTO role_permission (role_id, permission_id)
  SELECT '00000000-0000-7000-8000-000000000201', id FROM permission WHERE code IN
  ('profile.manage','address.manage','order.create','order.view.own',
   'order.cancel.own','payment.proof.upload','package.view.own',
   'delivery.view.own','delivery.schedule.own');

-- Admin holds everything, including the new own-scoped ones, so an admin can
-- reproduce what a customer sees when supporting them.
INSERT INTO role_permission (role_id, permission_id)
  SELECT '00000000-0000-7000-8000-000000000206', id FROM permission
   WHERE code IN ('profile.manage','address.manage','order.create','order.view.own',
                  'order.cancel.own','payment.proof.upload','package.view.own',
                  'delivery.view.own','delivery.schedule.own')
     AND NOT EXISTS (
       SELECT 1 FROM role_permission rp
        WHERE rp.role_id = '00000000-0000-7000-8000-000000000206'
          AND rp.permission_id = permission.id);

-- Staff act on customers' behalf (PROMPT §6.2.4: "staff can pick slots on the
-- customer's behalf"), so they need the scheduling capability too.
INSERT INTO role_permission (role_id, permission_id)
  SELECT '00000000-0000-7000-8000-000000000202', id FROM permission
   WHERE code IN ('address.manage','delivery.schedule.own','order.create');
