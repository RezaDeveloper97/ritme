<!DOCTYPE html>
<html lang="fa" dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="csrf-token" content="{{ csrf_token() }}">
    <title>صفحه تست - سیستم پیام‌رسانی</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://cdn.jsdelivr.net/gh/AliR4M/vazirmatn/Vazirmatn-font-face.css" rel="stylesheet" type="text/css" />
    <style>
        body { font-family: 'Vazirmatn', sans-serif; }
        .tab-active { border-bottom: 3px solid #8b5cf6; background: #f5f3ff; }
        .data-card { background: white; border-radius: 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.08); overflow: hidden; margin-bottom: 12px; }
        .data-card-header { padding: 12px 16px; font-weight: bold; font-size: 14px; border-bottom: 1px solid #f0f0f0; }
        .data-card-header.success { background: linear-gradient(135deg, #dcfce7, #bbf7d0); color: #166534; }
        .data-card-header.error { background: linear-gradient(135deg, #fee2e2, #fecaca); color: #991b1b; }
        .data-card-header.info { background: linear-gradient(135deg, #dbeafe, #bfdbfe); color: #1e40af; }
        .data-card-header.warning { background: linear-gradient(135deg, #fef3c7, #fde68a); color: #92400e; }
        .data-card-header.purple { background: linear-gradient(135deg, #f3e8ff, #e9d5ff); color: #7c3aed; }
        .data-card-header.pink { background: linear-gradient(135deg, #fce7f3, #fbcfe8); color: #db2777; }
        .data-card-header.teal { background: linear-gradient(135deg, #ccfbf1, #99f6e4); color: #0d9488; }
        .data-card-header.orange { background: linear-gradient(135deg, #ffedd5, #fed7aa); color: #ea580c; }
        .data-card-content { padding: 16px; }
        .data-row { display: flex; padding: 10px 0; border-bottom: 1px solid #f5f5f5; }
        .data-row:last-child { border-bottom: none; }
        .data-label { font-weight: 600; color: #6b7280; min-width: 160px; font-size: 13px; }
        .data-value { flex: 1; color: #1f2937; font-size: 14px; word-break: break-word; }
        .data-value.true { color: #059669; font-weight: bold; }
        .data-value.false { color: #dc2626; }
        .data-value.null { color: #9ca3af; font-style: italic; }
        .data-value.number { color: #7c3aed; font-weight: 600; }
        .badge { display: inline-block; padding: 4px 12px; border-radius: 20px; font-size: 12px; font-weight: bold; margin: 2px; }
        .badge-success { background: #dcfce7; color: #166534; }
        .badge-error { background: #fee2e2; color: #991b1b; }
        .badge-warning { background: #fef3c7; color: #92400e; }
        .badge-info { background: #dbeafe; color: #1e40af; }
        .badge-purple { background: #f3e8ff; color: #7c3aed; }
        .badge-pink { background: #fce7f3; color: #db2777; }
        .badge-teal { background: #ccfbf1; color: #0d9488; }
        .section-title { font-size: 13px; font-weight: bold; color: #9ca3af; text-transform: uppercase; letter-spacing: 0.5px; margin: 16px 0 8px 0; padding-bottom: 8px; border-bottom: 2px solid #f0f0f0; }
        .list-item { background: #f9fafb; border-radius: 8px; padding: 12px; margin-bottom: 8px; border-right: 4px solid #8b5cf6; }
        .response-container { background: linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%); border-radius: 16px; padding: 20px; min-height: 400px; max-height: 700px; overflow-y: auto; }
        .view-toggle { display: flex; gap: 8px; margin-bottom: 12px; }
        .view-toggle button { padding: 6px 16px; border-radius: 20px; font-size: 12px; font-weight: 600; border: 2px solid #e5e7eb; background: white; color: #6b7280; cursor: pointer; transition: all 0.2s; }
        .view-toggle button.active { background: #8b5cf6; border-color: #8b5cf6; color: white; }
        .json-raw { font-family: 'JetBrains Mono', 'Fira Code', monospace; font-size: 12px; direction: ltr; text-align: left; background: #1e293b; color: #e2e8f0; padding: 16px; border-radius: 12px; white-space: pre-wrap; word-break: break-all; }
        .phase-badge { padding: 8px 16px; border-radius: 8px; font-weight: bold; display: inline-block; }
        .phase-menstruation { background: #fee2e2; color: #991b1b; }
        .phase-follicular { background: #dbeafe; color: #1e40af; }
        .phase-ovulation { background: #dcfce7; color: #166534; }
        .phase-luteal { background: #fef3c7; color: #92400e; }
        .mode-cycle { background: #dbeafe; color: #1e40af; }
        .mode-pregnancy { background: #fce7f3; color: #db2777; }
    </style>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto px-4 py-6 max-w-7xl">
        <!-- Header -->
        <div class="bg-gradient-to-r from-violet-500 to-purple-600 rounded-2xl p-6 mb-6 text-white shadow-lg">
            <h1 class="text-3xl font-bold mb-2">صفحه تست سیستم پیام‌رسانی</h1>
            <p class="opacity-90">سیستم یکپارچه پیام‌رسانی برای سیکل و بارداری</p>
            <div class="mt-4 flex flex-wrap gap-4">
                <div class="bg-white/20 rounded-lg p-3">
                    <span class="text-sm">شماره تست: </span>
                    <span class="font-mono font-bold">09123456789</span>
                </div>
                <a href="/test-page" class="bg-white/20 hover:bg-white/30 rounded-lg p-3 transition">
                    <span class="text-sm">تست پروفایل</span>
                </a>
                <a href="/test-pregnancy" class="bg-white/20 hover:bg-white/30 rounded-lg p-3 transition">
                    <span class="text-sm">تست بارداری</span>
                </a>
            </div>
        </div>

        <!-- Mode Status -->
        <div id="modeStatus" class="bg-white rounded-xl shadow-lg p-4 mb-6">
            <div class="flex items-center justify-between">
                <div>
                    <span class="text-gray-500 text-sm">حالت فعلی:</span>
                    <span id="currentMode" class="mr-2 font-bold">در حال بارگذاری...</span>
                </div>
                <div id="modeDetails" class="flex gap-2"></div>
            </div>
        </div>

        <!-- Tabs -->
        <div class="bg-white rounded-t-xl shadow-lg overflow-hidden">
            <div class="flex border-b overflow-x-auto">
                <button onclick="switchTab('messages')" id="tab-messages" class="px-6 py-4 font-bold text-gray-700 hover:bg-gray-50 transition whitespace-nowrap tab-active">
                    پیام‌های روزانه
                </button>
                <button onclick="switchTab('settings')" id="tab-settings" class="px-6 py-4 font-bold text-gray-700 hover:bg-gray-50 transition whitespace-nowrap">
                    تنظیمات
                </button>
                <button onclick="switchTab('enums')" id="tab-enums" class="px-6 py-4 font-bold text-gray-700 hover:bg-gray-50 transition whitespace-nowrap">
                    Enum ها
                </button>
            </div>
        </div>

        <!-- Tab Contents -->
        <div class="bg-white rounded-b-xl shadow-lg p-6">
            <!-- Messages Tab -->
            <div id="content-messages" class="tab-content">
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">دریافت پیام‌های روزانه</h3>

                        <div class="mb-4">
                            <label class="block text-sm font-medium text-gray-700 mb-1">تاریخ</label>
                            <input type="date" id="message_date" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                        </div>

                        <div class="mb-4">
                            <label class="block text-sm font-medium text-gray-700 mb-1">حالت (اختیاری)</label>
                            <select id="force_mode" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                <option value="">تشخیص خودکار</option>
                                <option value="cycle">سیکل</option>
                                <option value="pregnancy">بارداری</option>
                            </select>
                        </div>

                        <button onclick="getDailyMessages()" class="w-full bg-violet-500 hover:bg-violet-600 text-white font-bold py-3 px-6 rounded-lg transition mb-4">
                            دریافت پیام‌ها
                        </button>

                        <div class="mb-4">
                            <h4 class="font-medium text-gray-700 mb-2">انتخاب سریع</h4>
                            <div class="grid grid-cols-4 gap-2">
                                <button onclick="setDateOffset(-7)" class="py-2 px-3 bg-gray-100 hover:bg-violet-100 rounded-lg text-sm font-medium transition">۷ روز قبل</button>
                                <button onclick="setDateOffset(-3)" class="py-2 px-3 bg-gray-100 hover:bg-violet-100 rounded-lg text-sm font-medium transition">۳ روز قبل</button>
                                <button onclick="setDateOffset(0)" class="py-2 px-3 bg-gray-100 hover:bg-violet-100 rounded-lg text-sm font-medium transition">امروز</button>
                                <button onclick="setDateOffset(7)" class="py-2 px-3 bg-gray-100 hover:bg-violet-100 rounded-lg text-sm font-medium transition">۷ روز بعد</button>
                            </div>
                        </div>

                        <div class="bg-gray-50 rounded-lg p-4">
                            <h4 class="font-medium text-gray-700 mb-3">راهنمای لایه‌ها</h4>
                            <div class="text-sm text-gray-600 space-y-1">
                                <div><span class="badge badge-pink">Layer 1&2</span> پیام پایه + Override بر اساس علائم</div>
                                <div><span class="badge badge-teal">Layer 3</span> همبستگی علائم</div>
                                <div><span class="badge badge-warning">Layer 4</span> تشخیص الگو (پرمیوم)</div>
                                <div><span class="badge badge-success">Supplements</span> تغذیه، خواب، ورزش</div>
                            </div>
                        </div>
                    </div>

                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">پاسخ API</h3>
                        <div class="view-toggle">
                            <button onclick="setViewMode('messagesResponse', 'pretty')" class="active" data-view="pretty">نمایش خوانا</button>
                            <button onclick="setViewMode('messagesResponse', 'raw')" data-view="raw">JSON خام</button>
                        </div>
                        <div class="response-container" id="messagesResponse">
                            <div class="text-center text-gray-400 py-12">
                                <p>پاسخ API اینجا نمایش داده می‌شود</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Settings Tab -->
            <div id="content-settings" class="tab-content hidden">
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">تنظیمات پروفایل</h3>

                        <form id="settingsForm" class="space-y-4">
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">هدف کاربر</label>
                                <select name="user_goal" id="user_goal" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                    <option value="">انتخاب کنید</option>
                                    <option value="non_ttc">غیر TTC (پیگیری سیکل)</option>
                                    <option value="ttc">TTC (تلاش برای بارداری)</option>
                                </select>
                            </div>

                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">نوع اشتراک</label>
                                <select name="subscription_type" id="subscription_type" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                    <option value="">انتخاب کنید</option>
                                    <option value="free">رایگان</option>
                                    <option value="premium">پرمیوم</option>
                                </select>
                            </div>

                            <button type="submit" class="w-full bg-violet-500 hover:bg-violet-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                ذخیره تنظیمات
                            </button>
                        </form>

                        <div class="mt-6 bg-amber-50 border border-amber-200 rounded-lg p-4">
                            <h4 class="font-medium text-amber-800 mb-2">توجه</h4>
                            <p class="text-sm text-amber-700">
                                برای دریافت پیام‌های سیکل، تاریخ آخرین پریود در <a href="/test-page" class="underline">صفحه تست پروفایل</a> باید وارد شده باشد.
                                <br>
                                برای دریافت پیام‌های بارداری، آنبوردینگ بارداری در <a href="/test-pregnancy" class="underline">صفحه تست بارداری</a> باید تکمیل شده باشد.
                            </p>
                        </div>
                    </div>

                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">پاسخ API</h3>
                        <div class="view-toggle">
                            <button onclick="setViewMode('settingsResponse', 'pretty')" class="active" data-view="pretty">نمایش خوانا</button>
                            <button onclick="setViewMode('settingsResponse', 'raw')" data-view="raw">JSON خام</button>
                        </div>
                        <div class="response-container" id="settingsResponse">
                            <div class="text-center text-gray-400 py-12">
                                <p>پاسخ API اینجا نمایش داده می‌شود</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Enums Tab -->
            <div id="content-enums" class="tab-content hidden">
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">Enum های سیستم پیام</h3>
                        <button onclick="getEnums()" class="w-full bg-violet-500 hover:bg-violet-600 text-white font-bold py-3 px-6 rounded-lg transition">
                            دریافت Enum ها
                        </button>
                    </div>

                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">پاسخ API</h3>
                        <div class="view-toggle">
                            <button onclick="setViewMode('enumsResponse', 'pretty')" class="active" data-view="pretty">نمایش خوانا</button>
                            <button onclick="setViewMode('enumsResponse', 'raw')" data-view="raw">JSON خام</button>
                        </div>
                        <div class="response-container" id="enumsResponse">
                            <div class="text-center text-gray-400 py-12">
                                <p>پاسخ API اینجا نمایش داده می‌شود</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <script>
        const csrfToken = document.querySelector('meta[name="csrf-token"]').getAttribute('content');
        let responseData = {};
        let viewModes = {};

        async function apiCall(url, method = 'GET', body = null) {
            const options = {
                method,
                headers: {
                    'Content-Type': 'application/json',
                    'X-CSRF-TOKEN': csrfToken,
                    'Accept': 'application/json',
                },
            };
            if (body) options.body = JSON.stringify(body);
            const response = await fetch(url, options);
            return await response.json();
        }

        function escapeHtml(str) {
            return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
        }

        function formatMessagesPretty(data) {
            if (!data) return '<div class="text-gray-400 text-center py-8">داده‌ای موجود نیست</div>';

            let html = '';

            if (data.success !== undefined) {
                html += `<div class="data-card"><div class="data-card-header ${data.success ? 'success' : 'error'}">${data.success ? '&#10004; موفق' : '&#10008; خطا'}</div>
                    ${data.message ? `<div class="data-card-content"><p>${escapeHtml(data.message)}</p></div>` : ''}</div>`;
            }

            if (!data.data) return html || '<div class="text-gray-400 text-center py-8">داده‌ای موجود نیست</div>';
            const d = data.data;

            // Mode & Basic Info
            const modeClass = d.mode === 'pregnancy' ? 'mode-pregnancy' : 'mode-cycle';
            html += `<div class="data-card"><div class="data-card-header info">اطلاعات عمومی</div><div class="data-card-content">
                <div class="data-row"><span class="data-label">حالت</span><span class="phase-badge ${modeClass}">${d.mode === 'pregnancy' ? 'بارداری' : 'سیکل'}</span></div>
                <div class="data-row"><span class="data-label">تاریخ</span><span class="data-value">${d.date || 'نامشخص'}</span></div>
                <div class="data-row"><span class="data-label">هدف کاربر</span><span class="data-value">${d.user_goal === 'ttc' ? 'TTC' : 'غیر TTC'}</span></div>
                <div class="data-row"><span class="data-label">اشتراک</span><span class="data-value">${d.subscription_type === 'premium' ? 'پرمیوم' : 'رایگان'}</span></div>
            </div></div>`;

            // Context Info
            if (d.context_info) {
                const ci = d.context_info;
                if (d.mode === 'cycle') {
                    const phaseClass = 'phase-' + (ci.phase || 'follicular');
                    html += `<div class="data-card"><div class="data-card-header purple">اطلاعات سیکل</div><div class="data-card-content">
                        <div class="text-center mb-4"><span class="phase-badge ${phaseClass}">${ci.phase_label || ci.phase || 'نامشخص'}</span></div>
                        <div class="data-row"><span class="data-label">زیرفاز</span><span class="data-value">${ci.subphase_label || ci.subphase || '-'}</span></div>
                        <div class="data-row"><span class="data-label">روز سیکل</span><span class="data-value number">${ci.cycle_day || '?'}</span></div>
                        <div class="data-row"><span class="data-label">پنجره باروری</span><span class="data-value ${ci.is_fertile_window ? 'true' : 'false'}">${ci.is_fertile_window ? '&#10004; بله' : '&#10008; خیر'}</span></div>
                        <div class="data-row"><span class="data-label">پنجره PMS</span><span class="data-value ${ci.is_pms_window ? 'true' : 'false'}">${ci.is_pms_window ? '&#10004; بله' : '&#10008; خیر'}</span></div>
                    </div></div>`;
                } else {
                    html += `<div class="data-card"><div class="data-card-header pink">اطلاعات بارداری</div><div class="data-card-content">
                        <div class="text-center mb-4"><span class="phase-badge mode-pregnancy">${ci.gestational_age || `هفته ${ci.week}`}</span></div>
                        <div class="data-row"><span class="data-label">سه‌ماهه</span><span class="data-value number">${ci.trimester || '?'}</span></div>
                        <div class="data-row"><span class="data-label">موعد زایمان</span><span class="data-value">${ci.due_date || '-'}</span></div>
                    </div></div>`;
                }
            }

            // Primary Message
            if (d.primary_message) {
                const pm = d.primary_message;
                html += `<div class="data-card"><div class="data-card-header pink">پیام اصلی (Layer 1&2)</div><div class="data-card-content">
                    ${pm.has_override ? '<div class="mb-2"><span class="badge badge-warning">Override فعال</span></div>' : ''}
                    ${pm.short_message ? `<div class="data-row"><span class="data-label">پیام کوتاه</span><span class="data-value">${escapeHtml(pm.short_message)}</span></div>` : ''}
                    ${pm.long_message ? `<div class="data-row"><span class="data-label">پیام بلند</span><span class="data-value">${escapeHtml(pm.long_message)}</span></div>` : ''}
                    ${pm.action_suggestion ? `<div class="data-row"><span class="data-label">پیشنهاد</span><span class="data-value">${escapeHtml(pm.action_suggestion)}</span></div>` : ''}
                    ${pm.fertility_info ? `<div class="data-row"><span class="data-label">وضعیت باروری</span><span class="data-value">${escapeHtml(pm.fertility_info)}</span></div>` : ''}
                    ${pm.dos?.length ? `<div class="data-row"><span class="data-label">بایدها</span><span class="data-value">${pm.dos.map(d => `<span class="badge badge-success">${escapeHtml(d)}</span>`).join('')}</span></div>` : ''}
                    ${pm.donts?.length ? `<div class="data-row"><span class="data-label">نبایدها</span><span class="data-value">${pm.donts.map(d => `<span class="badge badge-error">${escapeHtml(d)}</span>`).join('')}</span></div>` : ''}
                    ${pm.ttc_tips?.length ? `<div class="data-row"><span class="data-label">نکات TTC</span><span class="data-value">${pm.ttc_tips.map(t => `<span class="badge badge-pink">${escapeHtml(t)}</span>`).join('')}</span></div>` : ''}
                    ${pm.override_message ? `<div class="section-title">Override</div><div class="list-item"><p>${escapeHtml(pm.override_message)}</p>${pm.override_action ? `<p class="text-sm text-gray-500 mt-1">اقدام: ${escapeHtml(pm.override_action)}</p>` : ''}</div>` : ''}
                </div></div>`;
            }

            // Correlations
            if (d.correlations?.length) {
                html += `<div class="data-card"><div class="data-card-header teal">همبستگی‌ها (Layer 3)</div><div class="data-card-content">
                    ${d.correlations.map(c => `<div class="list-item">
                        <div class="flex justify-between mb-2"><span class="badge badge-teal">${c.type || 'correlation'}</span>${c.is_premium_only ? '<span class="badge badge-warning">پرمیوم</span>' : ''}</div>
                        ${c.insight_message ? `<p class="text-sm text-gray-700 mb-1">${escapeHtml(c.insight_message)}</p>` : ''}
                        ${c.action ? `<p class="text-xs text-gray-500">اقدام: ${escapeHtml(c.action)}</p>` : ''}
                    </div>`).join('')}
                </div></div>`;
            }

            // Patterns
            if (d.patterns?.length) {
                html += `<div class="data-card"><div class="data-card-header orange">الگوها (Layer 4 - Premium)</div><div class="data-card-content">
                    ${d.patterns.map(p => `<div class="list-item">
                        <div class="flex justify-between mb-2"><span class="badge badge-warning">${p.pattern_type || 'pattern'}</span><span class="badge ${p.alert_level === 'warning' ? 'badge-error' : 'badge-info'}">${p.alert_level || 'info'}</span></div>
                        ${p.message ? `<p class="text-sm text-gray-700 mb-1">${escapeHtml(p.message)}</p>` : ''}
                        ${p.recommendation ? `<p class="text-xs text-gray-500">توصیه: ${escapeHtml(p.recommendation)}</p>` : ''}
                    </div>`).join('')}
                </div></div>`;
            } else if (d.subscription_type === 'free') {
                html += `<div class="data-card"><div class="data-card-header orange">الگوها (Layer 4)</div><div class="data-card-content">
                    <div class="text-center text-gray-400 py-4">برای دسترسی به تشخیص الگو، اشتراک پرمیوم نیاز است</div>
                </div></div>`;
            }

            // Supplements
            if (d.supplements) {
                const s = d.supplements;
                html += `<div class="data-card"><div class="data-card-header success">نکات تکمیلی</div><div class="data-card-content">`;

                if (s.nutrition) {
                    html += `<div class="section-title">تغذیه</div>
                        ${s.nutrition.focus ? `<p class="text-sm text-gray-700 mb-2">${escapeHtml(s.nutrition.focus)}</p>` : ''}
                        ${s.nutrition.foods?.length ? `<div class="mb-2">${s.nutrition.foods.map(f => `<span class="badge badge-success">${escapeHtml(f)}</span>`).join('')}</div>` : ''}
                        ${s.nutrition.avoid?.length ? `<div class="mb-2">اجتناب: ${s.nutrition.avoid.map(a => `<span class="badge badge-error">${escapeHtml(a)}</span>`).join('')}</div>` : ''}`;
                }

                if (s.sleep) {
                    html += `<div class="section-title">خواب</div>
                        ${s.sleep.recommended_hours ? `<p class="text-sm text-gray-700 mb-2">ساعات توصیه شده: <strong>${s.sleep.recommended_hours}</strong></p>` : ''}
                        ${s.sleep.tips?.length ? `<div class="mb-2">${s.sleep.tips.map(t => `<span class="badge badge-info">${escapeHtml(t)}</span>`).join('')}</div>` : ''}`;
                }

                if (s.exercise) {
                    html += `<div class="section-title">ورزش</div>
                        ${s.exercise.intensity ? `<p class="text-sm text-gray-700 mb-2">شدت: <strong>${s.exercise.intensity}</strong></p>` : ''}
                        ${s.exercise.recommended?.length ? `<div class="mb-2">${s.exercise.recommended.map(r => `<span class="badge badge-purple">${escapeHtml(r)}</span>`).join('')}</div>` : ''}`;
                }

                html += `</div></div>`;
            }

            return html;
        }

        function formatPretty(data) {
            if (!data?.data) return '<div class="text-gray-400 text-center py-8">داده‌ای موجود نیست</div>';
            let html = `<div class="data-card"><div class="data-card-header ${data.success ? 'success' : 'error'}">${data.success ? '&#10004; موفق' : '&#10008; خطا'}</div>
                ${data.message ? `<div class="data-card-content"><p>${escapeHtml(data.message)}</p></div>` : ''}</div>`;
            html += `<div class="data-card"><div class="data-card-header info">اطلاعات</div><div class="data-card-content">`;
            for (const [key, value] of Object.entries(data.data)) {
                if (typeof value === 'object' && value !== null) {
                    html += `<div class="section-title">${key}</div><pre class="json-raw">${escapeHtml(JSON.stringify(value, null, 2))}</pre>`;
                } else {
                    html += `<div class="data-row"><span class="data-label">${key}</span><span class="data-value">${escapeHtml(String(value))}</span></div>`;
                }
            }
            html += `</div></div>`;
            return html;
        }

        function formatRaw(data) { return `<pre class="json-raw">${escapeHtml(JSON.stringify(data, null, 2))}</pre>`; }

        function setViewMode(elementId, mode) {
            viewModes[elementId] = mode;
            const container = document.getElementById(elementId).parentElement;
            container.querySelectorAll('.view-toggle button').forEach(btn => btn.classList.toggle('active', btn.dataset.view === mode));
            if (responseData[elementId]) displayResponse(elementId, responseData[elementId]);
        }

        function displayResponse(elementId, data) {
            responseData[elementId] = data;
            const mode = viewModes[elementId] || 'pretty';
            const container = document.getElementById(elementId);
            container.innerHTML = mode === 'pretty' ? (elementId === 'messagesResponse' ? formatMessagesPretty(data) : formatPretty(data)) : formatRaw(data);
        }

        function switchTab(tab) {
            document.querySelectorAll('.tab-content').forEach(el => el.classList.add('hidden'));
            document.querySelectorAll('[id^="tab-"]').forEach(el => el.classList.remove('tab-active'));
            document.getElementById(`content-${tab}`).classList.remove('hidden');
            document.getElementById(`tab-${tab}`).classList.add('tab-active');
        }

        function setDateOffset(days) {
            const date = new Date();
            date.setDate(date.getDate() + days);
            document.getElementById('message_date').value = date.toISOString().split('T')[0];
        }

        async function getDailyMessages() {
            const date = document.getElementById('message_date').value;
            const mode = document.getElementById('force_mode').value;
            let url = '/test-message/daily';
            const params = [];
            if (date) params.push(`date=${date}`);
            if (mode) params.push(`mode=${mode}`);
            if (params.length) url += '?' + params.join('&');
            const result = await apiCall(url);
            displayResponse('messagesResponse', result);
        }

        async function updateProfile() {
            const data = {
                user_goal: document.getElementById('user_goal').value,
                subscription_type: document.getElementById('subscription_type').value,
            };
            const result = await apiCall('/test-message/profile', 'POST', data);
            displayResponse('settingsResponse', result);
            loadModeStatus();
        }

        async function getEnums() {
            const result = await apiCall('/test-message/enums');
            displayResponse('enumsResponse', result);
        }

        async function loadModeStatus() {
            const result = await apiCall('/test-message/mode');
            const modeEl = document.getElementById('currentMode');
            const detailsEl = document.getElementById('modeDetails');

            if (result.success) {
                const d = result.data;
                const modeClass = d.mode === 'pregnancy' ? 'mode-pregnancy' : 'mode-cycle';
                modeEl.innerHTML = `<span class="phase-badge ${modeClass}">${d.mode_label || d.mode}</span>`;

                let details = '';
                if (d.user_goal) details += `<span class="badge ${d.is_ttc ? 'badge-pink' : 'badge-info'}">${d.user_goal_label || d.user_goal}</span>`;
                if (d.subscription_type) details += `<span class="badge ${d.is_premium ? 'badge-warning' : 'badge-info'}">${d.subscription_type_label || d.subscription_type}</span>`;
                detailsEl.innerHTML = details;

                if (d.user_goal) document.getElementById('user_goal').value = d.user_goal;
                if (d.subscription_type) document.getElementById('subscription_type').value = d.subscription_type;
            } else {
                modeEl.textContent = 'خطا در بارگذاری';
            }
        }

        document.getElementById('settingsForm').addEventListener('submit', async (e) => { e.preventDefault(); await updateProfile(); });

        document.getElementById('message_date').value = new Date().toISOString().split('T')[0];
        loadModeStatus();
    </script>
</body>
</html>
