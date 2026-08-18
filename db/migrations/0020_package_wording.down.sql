UPDATE package
   SET description = meal_credits || ' kredit makan, berlaku ' || validity_days || ' hari.'
 WHERE description LIKE '% hari makan sehat.';
