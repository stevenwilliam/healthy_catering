-- The new marketing pages' copy, editable in the back office.
--
-- Same model as the hero (0017): Indonesian is the source, English and Chinese
-- are seeded as hand-written overrides so the translator does not overwrite
-- them on the first edit. The templates read these through `c`, which falls
-- back to the compiled catalogue, so a missing row here is invisible rather
-- than a blank page.
INSERT INTO public_content (key, locale, value, is_override, source_hash) VALUES
 ('about.lede', 'id', 'Kami memasak makanan sehat harian di Jakarta dan mengantarnya dari dapur yang paling dekat dengan Anda.', FALSE, ''),
 ('about.body', 'id', 'Evermore berawal dari satu pertanyaan sederhana: bagaimana caranya makan sehat setiap hari tanpa harus memasak sendiri. Kami menyusun menu bersama ahli gizi, memasaknya pagi hari di beberapa dapur di Jakarta, dan mengantarnya ke rumah atau kantor Anda pada hari yang sama. Setiap porsi mencantumkan kalori dan proteinnya, karena Anda berhak tahu apa yang Anda makan.', FALSE, ''),
 ('career.lede', 'id', 'Kami sedang tumbuh, dan kami mencari orang yang peduli pada makanan yang baik.', FALSE, ''),
 ('career.body', 'id', 'Kami membuka kesempatan untuk juru masak, staf dapur, kurir, dan tim layanan pelanggan. Kirimkan CV Anda beserta posisi yang diminati, dan ceritakan sedikit mengapa Anda ingin bergabung. Kami membalas setiap lamaran yang masuk.', FALSE, ''),
 ('contact.lede', 'id', 'Ada pertanyaan tentang menu, pesanan kantor, atau area pengantaran? Kami senang membantu.', FALSE, '')
ON CONFLICT (key, locale) DO NOTHING;

INSERT INTO public_content (key, locale, value, is_override, source_hash)
SELECT v.key, v.locale, v.value, TRUE,
       encode(sha256(convert_to(src.value, 'UTF8')), 'hex')
  FROM (VALUES
    ('about.lede', 'en', 'We cook healthy daily meals in Jakarta and deliver them from the kitchen closest to you.'),
    ('about.lede', 'zh', '我们在雅加达制作每日健康餐，并由离您最近的厨房配送。'),
    ('about.body', 'en', 'Evermore started from one plain question: how do you eat well every day without having to cook. We plan the menu with nutritionists, cook it each morning across several kitchens in Jakarta, and deliver it to your home or office the same day. Every portion lists its calories and protein, because you should know what you are eating.'),
    ('about.body', 'zh', 'Evermore 源于一个朴素的问题：如何在不亲自下厨的情况下每天吃得健康。我们与营养师共同制定菜单，每天清晨在雅加达的多个厨房烹制，并于当天送达您的家中或办公室。每一份餐点都标明热量与蛋白质含量——您有权知道自己吃的是什么。'),
    ('career.lede', 'en', 'We are growing, and we are looking for people who care about good food.'),
    ('career.lede', 'zh', '我们正在成长，并寻找真正在意好食物的伙伴。'),
    ('career.body', 'en', 'We have openings for cooks, kitchen staff, couriers and customer-service team members. Send us your CV with the role you are interested in, and a few words on why you want to join. We reply to every application we receive.'),
    ('career.body', 'zh', '我们正在招聘厨师、厨房员工、配送员以及客服团队成员。请发送您的简历，注明意向职位，并简要说明您希望加入的原因。我们会回复每一份收到的申请。'),
    ('contact.lede', 'en', 'Questions about the menu, an office order, or where we deliver? We are glad to help.'),
    ('contact.lede', 'zh', '对菜单、企业订餐或配送范围有疑问？我们很乐意为您解答。')
  ) AS v(key, locale, value)
  JOIN public_content src ON src.key = v.key AND src.locale = 'id'
ON CONFLICT (key, locale) DO NOTHING;
