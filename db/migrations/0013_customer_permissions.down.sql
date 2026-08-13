DELETE FROM role_permission WHERE permission_id IN (
  SELECT id FROM permission WHERE code IN
  ('profile.manage','address.manage','order.create','order.view.own',
   'order.cancel.own','payment.proof.upload','package.view.own',
   'delivery.view.own','delivery.schedule.own'));
DELETE FROM permission WHERE code IN
  ('profile.manage','address.manage','order.create','order.view.own',
   'order.cancel.own','payment.proof.upload','package.view.own',
   'delivery.view.own','delivery.schedule.own');
