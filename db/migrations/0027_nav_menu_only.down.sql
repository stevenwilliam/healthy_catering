UPDATE nav_item SET is_visible = TRUE
 WHERE key IN ('category','pricelist','benefits','contact','about','career');
DELETE FROM nav_item WHERE key = 'menu';
