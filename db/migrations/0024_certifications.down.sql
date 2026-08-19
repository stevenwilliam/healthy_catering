DELETE FROM public_content
 WHERE key IN ('cert.heading','cert.halal_note','cert.haccp_note','cert.iso_note');
DELETE FROM sys_parameters WHERE key = 'public.certifications_enabled';
