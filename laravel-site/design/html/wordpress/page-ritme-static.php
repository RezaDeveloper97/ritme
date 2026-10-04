<?php
/**
 * Template Name: Ritme Static Page
 * Description: صفحه‌های استاتیک ریتمی را از پوشه /ritme-static/ در ریشه سایت می‌خواند و نمایش می‌دهد.
 * نصب: این فایل را در پوشه چایلد تم بگذارید، پوشه ritme-site را با نام ritme-static در ریشه وردپرس آپلود کنید،
 * سپس برای هر صفحه وردپرس (مثلاً نامک cycle) این قالب را انتخاب کنید. نامک صفحه = نام فایل HTML.
 */
$slug = get_post_field( 'post_name', get_post() );
$file = ABSPATH . 'ritme-static/' . preg_replace( '/[^a-z0-9\-]/', '', $slug ) . '.html';
if ( $slug === 'home' || is_front_page() ) { $file = ABSPATH . 'ritme-static/index.html'; }
if ( ! file_exists( $file ) ) { status_header( 404 ); echo 'صفحه استاتیک پیدا نشد: ' . esc_html( basename( $file ) ); exit; }
$html = file_get_contents( $file );
// مسیر assets را مطلق کن
$html = str_replace( array( 'href="assets/', 'src="assets/' ), array( 'href="/ritme-static/assets/', 'src="/ritme-static/assets/' ), $html );
// لینک‌های داخلی .html → پیوندهای وردپرس (اگر صفحه‌ها با همان نامک ساخته شده باشند)
$html = preg_replace( '/href="([a-z0-9\-]+)\.html"/', 'href="/$1/"', $html );
$html = str_replace( 'href="/index/"', 'href="/"', $html );
echo $html; // فایل کامل HTML است؛ هدر/فوتر تم لود نمی‌شود (عمداً).
