DELETE FROM sys_parameters WHERE key IN (
  'mail.host','mail.port','mail.username','mail.password','mail.from_email',
  'mail.from_name','mail.tls',
  'whatsapp.provider','whatsapp.waha_url','whatsapp.waha_session','whatsapp.waha_api_key',
  'order.min_qty','order.min_value_idr');
UPDATE sys_parameters SET value = '[{"max_km":5,"fee":0},{"max_km":10,"fee":15000},{"max_km":null,"fee":25000}]'
 WHERE key = 'delivery.fee_bands';
UPDATE sys_parameters SET value = '300000' WHERE key = 'delivery.free_above_idr';
DELETE FROM kitchen_operating_day WHERE weekday = 7;
