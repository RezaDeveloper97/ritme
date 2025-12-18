<!DOCTYPE html>
<html lang="fa" dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="csrf-token" content="{{ csrf_token() }}">
    <title>صفحه تست - ماتریکس پیام‌ها</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://cdn.jsdelivr.net/gh/AliR4M/vazirmatn/Vazirmatn-font-face.css" rel="stylesheet" type="text/css" />
    <style>
        body {
            font-family: 'Vazirmatn', sans-serif;
        }
        .tab-active {
            border-bottom: 3px solid #8b5cf6;
            background: #f5f3ff;
        }
        /* Pretty Card Styles */
        .data-card {
            background: white;
            border-radius: 12px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.08);
            overflow: hidden;
            margin-bottom: 12px;
        }
        .data-card-header {
            padding: 12px 16px;
            font-weight: bold;
            font-size: 14px;
            border-bottom: 1px solid #f0f0f0;
        }
        .data-card-header.success { background: linear-gradient(135deg, #dcfce7, #bbf7d0); color: #166534; }
        .data-card-header.error { background: linear-gradient(135deg, #fee2e2, #fecaca); color: #991b1b; }
        .data-card-header.info { background: linear-gradient(135deg, #dbeafe, #bfdbfe); color: #1e40af; }
        .data-card-header.warning { background: linear-gradient(135deg, #fef3c7, #fde68a); color: #92400e; }
        .data-card-header.purple { background: linear-gradient(135deg, #f3e8ff, #e9d5ff); color: #7c3aed; }
        .data-card-header.pink { background: linear-gradient(135deg, #fce7f3, #fbcfe8); color: #db2777; }
        .data-card-header.teal { background: linear-gradient(135deg, #ccfbf1, #99f6e4); color: #0d9488; }
        .data-card-header.orange { background: linear-gradient(135deg, #ffedd5, #fed7aa); color: #ea580c; }
        .data-card-content {
            padding: 16px;
        }
        .data-row {
            display: flex;
            padding: 10px 0;
            border-bottom: 1px solid #f5f5f5;
        }
        .data-row:last-child { border-bottom: none; }
        .data-label {
            font-weight: 600;
            color: #6b7280;
            min-width: 160px;
            font-size: 13px;
        }
        .data-value {
            flex: 1;
            color: #1f2937;
            font-size: 14px;
            word-break: break-word;
        }
        .data-value.true { color: #059669; font-weight: bold; }
        .data-value.false { color: #dc2626; }
        .data-value.null { color: #9ca3af; font-style: italic; }
        .data-value.number { color: #7c3aed; font-weight: 600; }
        .data-value.date { color: #0891b2; }
        .badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: bold;
        }
        .badge-success { background: #dcfce7; color: #166534; }
        .badge-error { background: #fee2e2; color: #991b1b; }
        .badge-warning { background: #fef3c7; color: #92400e; }
        .badge-info { background: #dbeafe; color: #1e40af; }
        .badge-purple { background: #f3e8ff; color: #7c3aed; }
        .badge-pink { background: #fce7f3; color: #db2777; }
        .badge-teal { background: #ccfbf1; color: #0d9488; }
        .section-title {
            font-size: 13px;
            font-weight: bold;
            color: #9ca3af;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin: 16px 0 8px 0;
            padding-bottom: 8px;
            border-bottom: 2px solid #f0f0f0;
        }
        .list-item {
            background: #f9fafb;
            border-radius: 8px;
            padding: 12px;
            margin-bottom: 8px;
            border-right: 4px solid #8b5cf6;
        }
        .list-item.ttc {
            border-right-color: #ec4899;
        }
        .list-item.correlation {
            border-right-color: #0891b2;
        }
        .list-item.pattern {
            border-right-color: #f59e0b;
        }
        .response-container {
            background: linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%);
            border-radius: 16px;
            padding: 20px;
            min-height: 400px;
            max-height: 700px;
            overflow-y: auto;
        }
        /* Raw JSON toggle */
        .view-toggle {
            display: flex;
            gap: 8px;
            margin-bottom: 12px;
        }
        .view-toggle button {
            padding: 6px 16px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 600;
            border: 2px solid #e5e7eb;
            background: white;
            color: #6b7280;
            cursor: pointer;
            transition: all 0.2s;
        }
        .view-toggle button.active {
            background: #8b5cf6;
            border-color: #8b5cf6;
            color: white;
        }
        .json-raw {
            font-family: 'JetBrains Mono', 'Fira Code', monospace;
            font-size: 12px;
            direction: ltr;
            text-align: left;
            background: #1e293b;
            color: #e2e8f0;
            padding: 16px;
            border-radius: 12px;
            white-space: pre-wrap;
            word-break: break-all;
        }
        .phase-badge {
            padding: 8px 16px;
            border-radius: 8px;
            font-weight: bold;
            display: inline-block;
        }
        .phase-menstruation { background: #fee2e2; color: #991b1b; }
        .phase-follicular { background: #dbeafe; color: #1e40af; }
        .phase-ovulation { background: #dcfce7; color: #166534; }
        .phase-luteal { background: #fef3c7; color: #92400e; }
    </style>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto px-4 py-6 max-w-7xl">
        <!-- Header -->
        <div class="bg-gradient-to-r from-violet-500 to-purple-600 rounded-2xl p-6 mb-6 text-white shadow-lg">
            <h1 class="text-3xl font-bold mb-2">صفحه تست ماتریکس پیام‌ها</h1>
            <p class="opacity-90">تست سیستم پیام‌رسانی هوشمند بر اساس فاز سیکل</p>
            <div class="mt-4 flex flex-wrap gap-4">
                <div class="bg-white/20 rounded-lg p-3">
                    <span class="text-sm">شماره تست: </span>
                    <span class="font-mono font-bold">09123456789</span>
                </div>
                <a href="/test-page" class="bg-white/20 hover:bg-white/30 rounded-lg p-3 transition">
                    <span class="text-sm">صفحه تست اصلی</span>
                </a>
                <a href="/test-pregnancy" class="bg-white/20 hover:bg-white/30 rounded-lg p-3 transition">
                    <span class="text-sm">صفحه تست بارداری</span>
                </a>
            </div>
        </div>

        <!-- Tabs -->
        <div class="bg-white rounded-t-xl shadow-lg overflow-hidden">
            <div class="flex border-b overflow-x-auto">
                <button onclick="switchTab('messages')" id="tab-messages" class="px-6 py-4 font-bold text-gray-700 hover:bg-gray-50 transition whitespace-nowrap tab-active">
                    پیام‌های ماتریکس
                </button>
                <button onclick="switchTab('settings')" id="tab-settings" class="px-6 py-4 font-bold text-gray-700 hover:bg-gray-50 transition whitespace-nowrap">
                    تنظیمات پروفایل
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
                    <!-- Request Form -->
                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">دریافت پیام‌های ماتریکس</h3>

                        <!-- Date Selector -->
                        <div class="mb-4">
                            <label class="block text-sm font-medium text-gray-700 mb-1">تاریخ</label>
                            <input type="date" id="message_date" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                        </div>

                        <!-- Current Profile Info -->
                        <div class="bg-gray-50 rounded-lg p-4 mb-4">
                            <h4 class="font-medium text-gray-700 mb-2">وضعیت فعلی پروفایل</h4>
                            <div id="profileStatus" class="text-sm text-gray-600">
                                در حال بارگذاری...
                            </div>
                        </div>

                        <button onclick="getMatrixMessages()" class="w-full bg-violet-500 hover:bg-violet-600 text-white font-bold py-3 px-6 rounded-lg transition mb-4">
                            دریافت پیام‌ها
                        </button>

                        <!-- Quick Date Navigation -->
                        <div class="mb-4">
                            <h4 class="font-medium text-gray-700 mb-2">انتخاب سریع</h4>
                            <div class="grid grid-cols-4 gap-2">
                                <button onclick="setDateOffset(-7)" class="py-2 px-3 bg-gray-100 hover:bg-violet-100 rounded-lg text-sm font-medium transition">
                                    ۷ روز قبل
                                </button>
                                <button onclick="setDateOffset(-3)" class="py-2 px-3 bg-gray-100 hover:bg-violet-100 rounded-lg text-sm font-medium transition">
                                    ۳ روز قبل
                                </button>
                                <button onclick="setDateOffset(0)" class="py-2 px-3 bg-gray-100 hover:bg-violet-100 rounded-lg text-sm font-medium transition">
                                    امروز
                                </button>
                                <button onclick="setDateOffset(7)" class="py-2 px-3 bg-gray-100 hover:bg-violet-100 rounded-lg text-sm font-medium transition">
                                    ۷ روز بعد
                                </button>
                            </div>
                        </div>

                        <!-- Legend -->
                        <div class="bg-gray-50 rounded-lg p-4">
                            <h4 class="font-medium text-gray-700 mb-3">راهنما</h4>
                            <div class="grid grid-cols-2 gap-2 text-sm">
                                <div class="flex items-center">
                                    <span class="w-3 h-3 rounded-full bg-red-400 ml-2"></span>
                                    <span>قاعدگی</span>
                                </div>
                                <div class="flex items-center">
                                    <span class="w-3 h-3 rounded-full bg-blue-400 ml-2"></span>
                                    <span>فولیکولار</span>
                                </div>
                                <div class="flex items-center">
                                    <span class="w-3 h-3 rounded-full bg-green-400 ml-2"></span>
                                    <span>تخمک‌گذاری</span>
                                </div>
                                <div class="flex items-center">
                                    <span class="w-3 h-3 rounded-full bg-amber-400 ml-2"></span>
                                    <span>لوتئال</span>
                                </div>
                            </div>
                            <hr class="my-3">
                            <div class="text-sm text-gray-600">
                                <div class="mb-1"><strong>Layer 1&2:</strong> پیام پایه بر اساس فاز</div>
                                <div class="mb-1"><strong>Layer 3:</strong> همبستگی علائم (رایگان/پرمیوم)</div>
                                <div><strong>Layer 4:</strong> تشخیص الگو (فقط پرمیوم)</div>
                            </div>
                        </div>
                    </div>

                    <!-- Response Display -->
                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">پاسخ API</h3>
                        <div class="view-toggle">
                            <button onclick="setViewMode('messagesResponse', 'pretty')" class="active" data-view="pretty">نمایش خوانا</button>
                            <button onclick="setViewMode('messagesResponse', 'raw')" data-view="raw">JSON خام</button>
                        </div>
                        <div class="response-container" id="messagesResponse">
                            <div class="text-center text-gray-400 py-12">
                                <svg class="w-16 h-16 mx-auto mb-4 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
                                </svg>
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
                                </select>
                                <p class="text-xs text-gray-500 mt-1">TTC = تلاش برای بارداری | Non-TTC = پیگیری عادی سیکل</p>
                            </div>

                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">نوع اشتراک</label>
                                <select name="subscription_type" id="subscription_type" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                    <option value="">انتخاب کنید</option>
                                </select>
                                <p class="text-xs text-gray-500 mt-1">Premium = دسترسی به Layer 4 (الگوها) و همبستگی‌های پیشرفته</p>
                            </div>

                            <button type="submit" class="w-full bg-violet-500 hover:bg-violet-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                ذخیره تنظیمات
                            </button>
                        </form>

                        <div class="mt-6">
                            <button onclick="getProfile()" class="w-full bg-blue-500 hover:bg-blue-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                دریافت پروفایل فعلی
                            </button>
                        </div>

                        <div class="mt-6 bg-amber-50 border border-amber-200 rounded-lg p-4">
                            <h4 class="font-medium text-amber-800 mb-2">توجه</h4>
                            <p class="text-sm text-amber-700">
                                برای دریافت پیام‌های ماتریکس، باید تاریخ آخرین پریود در صفحه تست اصلی وارد شده باشد.
                                <a href="/test-page" class="underline">برو به صفحه تست اصلی</a>
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
                        <h3 class="text-lg font-bold mb-4 text-gray-800">Enum های ماتریکس</h3>
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
        let enums = {};
        let responseData = {};
        let viewModes = {};

        // Persian translations for keys
        const keyTranslations = {
            // Common
            'success': 'موفقیت',
            'message': 'پیام',
            'data': 'اطلاعات',
            'date': 'تاریخ',

            // User
            'user_goal': 'هدف کاربر',
            'user_goal_label': 'برچسب هدف',
            'subscription_type': 'نوع اشتراک',
            'subscription_type_label': 'برچسب اشتراک',
            'is_ttc': 'تلاش برای بارداری',
            'is_premium': 'اشتراک پرمیوم',
            'last_period_start': 'شروع آخرین پریود',
            'cycle_length': 'طول سیکل',

            // Cycle Info
            'cycle_info': 'اطلاعات سیکل',
            'phase': 'فاز',
            'phase_label': 'نام فاز',
            'subphase': 'زیرفاز',
            'subphase_label': 'نام زیرفاز',
            'cycle_day': 'روز سیکل',
            'is_fertile_window': 'پنجره باروری',
            'is_pms_window': 'پنجره PMS',
            'estimated_ovulation_day': 'روز تخمینی تخمک‌گذاری',
            'cycle_length_used': 'طول سیکل استفاده شده',

            // Matrix Message
            'matrix_message': 'پیام ماتریکس',
            'override_type': 'نوع Override',
            'short_message': 'پیام کوتاه',
            'long_message': 'پیام بلند',
            'action_suggestion': 'پیشنهاد اقدام',
            'dos': 'بایدها',
            'donts': 'نبایدها',

            // Correlations
            'correlations': 'همبستگی‌ها',
            'type': 'نوع',
            'insight_message': 'پیام بینش',
            'action': 'اقدام',
            'is_premium_only': 'فقط پرمیوم',

            // Patterns
            'patterns': 'الگوها',
            'pattern_type': 'نوع الگو',
            'alert_level': 'سطح هشدار',
            'recommendation': 'توصیه',

            // Nutrition & Sleep
            'nutrition_sleep_tips': 'نکات تغذیه و خواب',
            'nutrition': 'تغذیه',
            'sleep': 'خواب',
            'exercise': 'ورزش',
            'tip': 'نکته',
            'foods': 'غذاها',
            'avoid': 'اجتناب کنید',
            'recommended_hours': 'ساعات توصیه شده',
            'tip_for_better_sleep': 'نکته برای خواب بهتر',
            'intensity': 'شدت',
            'exercises': 'ورزش‌ها',

            // Enums
            'phases': 'فازها',
            'subphases': 'زیرفازها',
            'user_goals': 'اهداف کاربر',
            'subscription_types': 'انواع اشتراک',
            'override_types': 'انواع Override',
            'value': 'مقدار',
            'label': 'برچسب',
            'description': 'توضیحات',
        };

        // Value translations
        const valueTranslations = {
            'true': 'بله',
            'false': 'خیر',
            'null': 'ندارد',
            'ttc': 'تلاش برای بارداری',
            'non_ttc': 'غیر TTC',
            'free': 'رایگان',
            'premium': 'پرمیوم',
            'menstruation': 'قاعدگی',
            'follicular': 'فولیکولار',
            'ovulation': 'تخمک‌گذاری',
            'luteal': 'لوتئال',
            'normal': 'عادی',
            'pain': 'درد',
            'mood_sad': 'غمگین',
            'mood_angry': 'عصبانی',
            'heavy_flow': 'خونریزی شدید',
            'low': 'کم',
            'moderate': 'متوسط',
            'high': 'زیاد',
            'warning': 'هشدار',
            'info': 'اطلاعاتی',
            'critical': 'بحرانی',
        };

        // Fetch helper
        async function apiCall(url, method = 'GET', body = null) {
            const options = {
                method,
                headers: {
                    'Content-Type': 'application/json',
                    'X-CSRF-TOKEN': csrfToken,
                    'Accept': 'application/json',
                },
            };
            if (body) {
                options.body = JSON.stringify(body);
            }
            const response = await fetch(url, options);
            return await response.json();
        }

        function escapeHtml(str) {
            return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
        }

        function translateKey(key) {
            return keyTranslations[key] || key;
        }

        function translateValue(value) {
            if (value === true) return 'بله';
            if (value === false) return 'خیر';
            if (value === null || value === undefined) return 'ندارد';
            const str = String(value);
            return valueTranslations[str] || str;
        }

        // Pretty format for messages display
        function formatMatrixPretty(data) {
            if (!data) return '<div class="text-gray-400 text-center py-8">داده‌ای موجود نیست</div>';

            let html = '';

            // Status card
            if (data.success !== undefined) {
                const isSuccess = data.success;
                html += `
                    <div class="data-card">
                        <div class="data-card-header ${isSuccess ? 'success' : 'error'}">
                            ${isSuccess ? '&#10004; عملیات موفق' : '&#10008; خطا'}
                        </div>
                        ${data.message ? `<div class="data-card-content"><p>${escapeHtml(data.message)}</p></div>` : ''}
                    </div>
                `;
            }

            if (!data.data) return html || '<div class="text-gray-400 text-center py-8">داده‌ای موجود نیست</div>';

            const d = data.data;

            // Date and User Info
            html += `
                <div class="data-card">
                    <div class="data-card-header info">اطلاعات عمومی</div>
                    <div class="data-card-content">
                        <div class="data-row">
                            <span class="data-label">تاریخ</span>
                            <span class="data-value date">${d.date || 'نامشخص'}</span>
                        </div>
                        <div class="data-row">
                            <span class="data-label">هدف کاربر</span>
                            <span class="data-value">${d.user_goal_label || translateValue(d.user_goal)}</span>
                        </div>
                        <div class="data-row">
                            <span class="data-label">نوع اشتراک</span>
                            <span class="data-value">${d.subscription_type_label || translateValue(d.subscription_type)}</span>
                        </div>
                    </div>
                </div>
            `;

            // Cycle Info
            if (d.cycle_info) {
                const ci = d.cycle_info;
                const phaseClass = 'phase-' + (ci.phase || 'follicular');
                html += `
                    <div class="data-card">
                        <div class="data-card-header purple">اطلاعات سیکل</div>
                        <div class="data-card-content">
                            <div class="text-center mb-4">
                                <span class="phase-badge ${phaseClass}">${ci.phase_label || translateValue(ci.phase)}</span>
                            </div>
                            <div class="data-row">
                                <span class="data-label">زیرفاز</span>
                                <span class="data-value">${ci.subphase_label || translateValue(ci.subphase)}</span>
                            </div>
                            <div class="data-row">
                                <span class="data-label">روز سیکل</span>
                                <span class="data-value number">${ci.cycle_day || '?'}</span>
                            </div>
                            <div class="data-row">
                                <span class="data-label">پنجره باروری</span>
                                <span class="data-value ${ci.is_fertile_window ? 'true' : 'false'}">${ci.is_fertile_window ? '&#10004; بله' : '&#10008; خیر'}</span>
                            </div>
                            <div class="data-row">
                                <span class="data-label">پنجره PMS</span>
                                <span class="data-value ${ci.is_pms_window ? 'true' : 'false'}">${ci.is_pms_window ? '&#10004; بله' : '&#10008; خیر'}</span>
                            </div>
                            ${ci.estimated_ovulation_day ? `
                            <div class="data-row">
                                <span class="data-label">روز تخمینی تخمک‌گذاری</span>
                                <span class="data-value number">${ci.estimated_ovulation_day}</span>
                            </div>` : ''}
                        </div>
                    </div>
                `;
            }

            // Matrix Message (Layer 1&2)
            if (d.matrix_message) {
                const mm = d.matrix_message;
                html += `
                    <div class="data-card">
                        <div class="data-card-header pink">پیام ماتریکس (Layer 1&2)</div>
                        <div class="data-card-content">
                            ${mm.override_type ? `
                            <div class="data-row">
                                <span class="data-label">نوع Override</span>
                                <span class="badge badge-purple">${translateValue(mm.override_type)}</span>
                            </div>` : ''}
                            ${mm.short_message ? `
                            <div class="data-row">
                                <span class="data-label">پیام کوتاه</span>
                                <span class="data-value">${escapeHtml(mm.short_message)}</span>
                            </div>` : ''}
                            ${mm.long_message ? `
                            <div class="data-row">
                                <span class="data-label">پیام بلند</span>
                                <span class="data-value">${escapeHtml(mm.long_message)}</span>
                            </div>` : ''}
                            ${mm.action_suggestion ? `
                            <div class="data-row">
                                <span class="data-label">پیشنهاد اقدام</span>
                                <span class="data-value">${escapeHtml(mm.action_suggestion)}</span>
                            </div>` : ''}
                            ${mm.dos && mm.dos.length > 0 ? `
                            <div class="data-row">
                                <span class="data-label">بایدها</span>
                                <span class="data-value">
                                    ${mm.dos.map(d => `<span class="badge badge-success ml-1 mb-1">${escapeHtml(d)}</span>`).join('')}
                                </span>
                            </div>` : ''}
                            ${mm.donts && mm.donts.length > 0 ? `
                            <div class="data-row">
                                <span class="data-label">نبایدها</span>
                                <span class="data-value">
                                    ${mm.donts.map(d => `<span class="badge badge-error ml-1 mb-1">${escapeHtml(d)}</span>`).join('')}
                                </span>
                            </div>` : ''}
                        </div>
                    </div>
                `;
            }

            // Correlations (Layer 3)
            if (d.correlations && d.correlations.length > 0) {
                html += `
                    <div class="data-card">
                        <div class="data-card-header teal">همبستگی‌ها (Layer 3)</div>
                        <div class="data-card-content">
                            ${d.correlations.map(c => `
                                <div class="list-item correlation">
                                    <div class="flex justify-between items-start mb-2">
                                        <span class="badge badge-teal">${translateValue(c.type)}</span>
                                        ${c.is_premium_only ? '<span class="badge badge-warning">پرمیوم</span>' : ''}
                                    </div>
                                    ${c.insight_message ? `<p class="text-sm text-gray-700 mb-1">${escapeHtml(c.insight_message)}</p>` : ''}
                                    ${c.action ? `<p class="text-xs text-gray-500"><strong>اقدام:</strong> ${escapeHtml(c.action)}</p>` : ''}
                                </div>
                            `).join('')}
                        </div>
                    </div>
                `;
            }

            // Patterns (Layer 4)
            if (d.patterns && d.patterns.length > 0) {
                html += `
                    <div class="data-card">
                        <div class="data-card-header orange">الگوها (Layer 4 - Premium)</div>
                        <div class="data-card-content">
                            ${d.patterns.map(p => `
                                <div class="list-item pattern">
                                    <div class="flex justify-between items-start mb-2">
                                        <span class="badge badge-warning">${translateValue(p.pattern_type)}</span>
                                        <span class="badge ${p.alert_level === 'warning' ? 'badge-error' : 'badge-info'}">${translateValue(p.alert_level)}</span>
                                    </div>
                                    ${p.message ? `<p class="text-sm text-gray-700 mb-1">${escapeHtml(p.message)}</p>` : ''}
                                    ${p.recommendation ? `<p class="text-xs text-gray-500"><strong>توصیه:</strong> ${escapeHtml(p.recommendation)}</p>` : ''}
                                </div>
                            `).join('')}
                        </div>
                    </div>
                `;
            } else if (d.subscription_type === 'free') {
                html += `
                    <div class="data-card">
                        <div class="data-card-header orange">الگوها (Layer 4)</div>
                        <div class="data-card-content">
                            <div class="text-center text-gray-400 py-4">
                                <p>برای دسترسی به تشخیص الگو، به اشتراک پرمیوم ارتقا دهید</p>
                            </div>
                        </div>
                    </div>
                `;
            }

            // Nutrition & Sleep Tips
            if (d.nutrition_sleep_tips) {
                const tips = d.nutrition_sleep_tips;
                html += `
                    <div class="data-card">
                        <div class="data-card-header success">نکات تغذیه، خواب و ورزش</div>
                        <div class="data-card-content">
                `;

                if (tips.nutrition) {
                    html += `
                        <div class="section-title">تغذیه</div>
                        ${tips.nutrition.tip ? `<p class="text-sm text-gray-700 mb-2">${escapeHtml(tips.nutrition.tip)}</p>` : ''}
                        ${tips.nutrition.foods && tips.nutrition.foods.length > 0 ? `
                        <div class="data-row">
                            <span class="data-label">غذاهای پیشنهادی</span>
                            <span class="data-value">${tips.nutrition.foods.map(f => `<span class="badge badge-success ml-1">${escapeHtml(f)}</span>`).join('')}</span>
                        </div>` : ''}
                        ${tips.nutrition.avoid && tips.nutrition.avoid.length > 0 ? `
                        <div class="data-row">
                            <span class="data-label">اجتناب کنید</span>
                            <span class="data-value">${tips.nutrition.avoid.map(a => `<span class="badge badge-error ml-1">${escapeHtml(a)}</span>`).join('')}</span>
                        </div>` : ''}
                    `;
                }

                if (tips.sleep) {
                    html += `
                        <div class="section-title">خواب</div>
                        ${tips.sleep.recommended_hours ? `
                        <div class="data-row">
                            <span class="data-label">ساعات توصیه شده</span>
                            <span class="data-value number">${tips.sleep.recommended_hours} ساعت</span>
                        </div>` : ''}
                        ${tips.sleep.tip_for_better_sleep ? `
                        <div class="data-row">
                            <span class="data-label">نکته</span>
                            <span class="data-value">${escapeHtml(tips.sleep.tip_for_better_sleep)}</span>
                        </div>` : ''}
                    `;
                }

                if (tips.exercise) {
                    html += `
                        <div class="section-title">ورزش</div>
                        ${tips.exercise.intensity ? `
                        <div class="data-row">
                            <span class="data-label">شدت پیشنهادی</span>
                            <span class="data-value">${translateValue(tips.exercise.intensity)}</span>
                        </div>` : ''}
                        ${tips.exercise.exercises && tips.exercise.exercises.length > 0 ? `
                        <div class="data-row">
                            <span class="data-label">ورزش‌های پیشنهادی</span>
                            <span class="data-value">${tips.exercise.exercises.map(e => `<span class="badge badge-info ml-1">${escapeHtml(e)}</span>`).join('')}</span>
                        </div>` : ''}
                        ${tips.exercise.tip ? `
                        <div class="data-row">
                            <span class="data-label">نکته</span>
                            <span class="data-value">${escapeHtml(tips.exercise.tip)}</span>
                        </div>` : ''}
                    `;
                }

                html += `
                        </div>
                    </div>
                `;
            }

            return html || '<div class="text-gray-400 text-center py-8">داده‌ای موجود نیست</div>';
        }

        // Generic pretty format
        function formatPretty(data, elementId) {
            if (!data) return '<div class="text-gray-400 text-center py-8">داده‌ای موجود نیست</div>';

            let html = '';

            // Status card
            if (data.success !== undefined) {
                const isSuccess = data.success;
                html += `
                    <div class="data-card">
                        <div class="data-card-header ${isSuccess ? 'success' : 'error'}">
                            ${isSuccess ? '&#10004; عملیات موفق' : '&#10008; خطا'}
                        </div>
                        ${data.message ? `<div class="data-card-content"><p>${escapeHtml(data.message)}</p></div>` : ''}
                    </div>
                `;
            }

            // Main data
            if (data.data) {
                html += renderObject(data.data, 'اطلاعات');
            }

            return html || '<div class="text-gray-400 text-center py-8">داده‌ای موجود نیست</div>';
        }

        function renderObject(obj, title = '') {
            if (obj === null || obj === undefined) return '';
            if (typeof obj !== 'object') return `<span>${translateValue(obj)}</span>`;

            let html = '';

            if (Array.isArray(obj)) {
                if (obj.length === 0) {
                    return `<div class="text-gray-400 text-sm">لیست خالی است</div>`;
                }
                html += `<div class="section-title">${title || 'لیست'} (${obj.length} مورد)</div>`;
                obj.forEach((item, index) => {
                    if (typeof item === 'object' && item !== null) {
                        html += `<div class="list-item">`;
                        html += renderObjectRows(item);
                        html += `</div>`;
                    } else {
                        html += `<div class="list-item"><span class="badge badge-info">${translateValue(item)}</span></div>`;
                    }
                });
                return html;
            }

            html += `<div class="data-card">`;
            if (title) html += `<div class="data-card-header info">${title}</div>`;
            html += `<div class="data-card-content">`;
            html += renderObjectRows(obj);
            html += `</div></div>`;

            return html;
        }

        function renderObjectRows(obj) {
            if (!obj || typeof obj !== 'object') return '';

            let html = '';
            for (const [key, value] of Object.entries(obj)) {
                const label = translateKey(key);

                if (Array.isArray(value)) {
                    html += `<div class="data-row">
                        <span class="data-label">${label}</span>
                        <span class="data-value">`;
                    if (value.length === 0) {
                        html += '<span class="null">ندارد</span>';
                    } else {
                        value.forEach(v => {
                            if (typeof v === 'object' && v !== null) {
                                html += `<div class="list-item">${renderObjectRows(v)}</div>`;
                            } else {
                                html += `<span class="badge badge-purple ml-1">${translateValue(v)}</span>`;
                            }
                        });
                    }
                    html += `</span></div>`;
                } else if (typeof value === 'object' && value !== null) {
                    html += `<div class="data-row">
                        <span class="data-label">${label}</span>
                        <span class="data-value">`;
                    html += renderObjectRows(value);
                    html += `</span></div>`;
                } else if (typeof value === 'boolean') {
                    html += `<div class="data-row">
                        <span class="data-label">${label}</span>
                        <span class="data-value ${value ? 'true' : 'false'}">
                            ${value ? '&#10004; بله' : '&#10008; خیر'}
                        </span>
                    </div>`;
                } else if (value === null || value === undefined) {
                    html += `<div class="data-row">
                        <span class="data-label">${label}</span>
                        <span class="data-value null">ندارد</span>
                    </div>`;
                } else if (typeof value === 'number') {
                    html += `<div class="data-row">
                        <span class="data-label">${label}</span>
                        <span class="data-value number">${value}</span>
                    </div>`;
                } else {
                    html += `<div class="data-row">
                        <span class="data-label">${label}</span>
                        <span class="data-value">${translateValue(value)}</span>
                    </div>`;
                }
            }
            return html;
        }

        // Raw JSON format
        function formatRaw(data) {
            return `<pre class="json-raw">${escapeHtml(JSON.stringify(data, null, 2))}</pre>`;
        }

        // View mode toggle
        function setViewMode(elementId, mode) {
            viewModes[elementId] = mode;

            // Update buttons
            const container = document.getElementById(elementId).parentElement;
            container.querySelectorAll('.view-toggle button').forEach(btn => {
                btn.classList.toggle('active', btn.dataset.view === mode);
            });

            // Re-render
            if (responseData[elementId]) {
                displayResponse(elementId, responseData[elementId]);
            }
        }

        function displayResponse(elementId, data) {
            responseData[elementId] = data;
            const mode = viewModes[elementId] || 'pretty';
            const container = document.getElementById(elementId);

            if (mode === 'pretty') {
                if (elementId === 'messagesResponse') {
                    container.innerHTML = formatMatrixPretty(data);
                } else {
                    container.innerHTML = formatPretty(data, elementId);
                }
            } else {
                container.innerHTML = formatRaw(data);
            }
        }

        // Tab switching
        function switchTab(tab) {
            document.querySelectorAll('.tab-content').forEach(el => el.classList.add('hidden'));
            document.querySelectorAll('[id^="tab-"]').forEach(el => el.classList.remove('tab-active'));
            document.getElementById(`content-${tab}`).classList.remove('hidden');
            document.getElementById(`tab-${tab}`).classList.add('tab-active');
        }

        // Initialize enums
        async function loadEnums() {
            const result = await apiCall('/test-matrix/enums');
            if (result.success) {
                enums = result.data;
                populateEnumSelects();
            }
        }

        function populateEnumSelects() {
            // User goals
            const userGoalSelect = document.getElementById('user_goal');
            enums.user_goals?.forEach(item => {
                const option = document.createElement('option');
                option.value = item.value;
                option.textContent = item.label;
                userGoalSelect.appendChild(option);
            });

            // Subscription types
            const subscriptionSelect = document.getElementById('subscription_type');
            enums.subscription_types?.forEach(item => {
                const option = document.createElement('option');
                option.value = item.value;
                option.textContent = item.label;
                subscriptionSelect.appendChild(option);
            });
        }

        // Date helpers
        function setDateOffset(days) {
            const date = new Date();
            date.setDate(date.getDate() + days);
            document.getElementById('message_date').value = date.toISOString().split('T')[0];
        }

        // API Functions
        async function getMatrixMessages() {
            const date = document.getElementById('message_date').value;
            let url = '/test-matrix/messages';
            if (date) {
                url += '?date=' + date;
            }
            const result = await apiCall(url);
            displayResponse('messagesResponse', result);
        }

        async function updateProfile() {
            const data = {
                user_goal: document.getElementById('user_goal').value,
                subscription_type: document.getElementById('subscription_type').value,
            };
            const result = await apiCall('/test-matrix/profile', 'POST', data);
            displayResponse('settingsResponse', result);
            loadProfileStatus();
        }

        async function getProfile() {
            const result = await apiCall('/test-matrix/profile');
            displayResponse('settingsResponse', result);
        }

        async function getEnums() {
            const result = await apiCall('/test-matrix/enums');
            displayResponse('enumsResponse', result);
        }

        async function loadProfileStatus() {
            const result = await apiCall('/test-matrix/profile');
            const container = document.getElementById('profileStatus');

            if (result.success) {
                const d = result.data;
                container.innerHTML = `
                    <div class="space-y-1">
                        <div><strong>هدف:</strong> ${d.user_goal_label || 'تنظیم نشده'}</div>
                        <div><strong>اشتراک:</strong> ${d.subscription_type_label || 'تنظیم نشده'}</div>
                        <div><strong>TTC:</strong> ${d.is_ttc ? '<span class="text-pink-600">بله</span>' : '<span class="text-gray-600">خیر</span>'}</div>
                        <div><strong>پرمیوم:</strong> ${d.is_premium ? '<span class="text-violet-600">بله</span>' : '<span class="text-gray-600">خیر</span>'}</div>
                        ${d.last_period_start ? `<div><strong>آخرین پریود:</strong> ${d.last_period_start}</div>` : '<div class="text-amber-600">آخرین پریود تنظیم نشده!</div>'}
                    </div>
                `;

                // Set current values in form
                if (d.user_goal) {
                    document.getElementById('user_goal').value = d.user_goal;
                }
                if (d.subscription_type) {
                    document.getElementById('subscription_type').value = d.subscription_type;
                }
            } else {
                container.innerHTML = `<span class="text-red-500">${result.message || 'خطا در بارگذاری'}</span>`;
            }
        }

        // Form submission
        document.getElementById('settingsForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            await updateProfile();
        });

        // Initialize
        function setDefaultDates() {
            const today = new Date().toISOString().split('T')[0];
            document.getElementById('message_date').value = today;
        }

        setDefaultDates();
        loadEnums();
        loadProfileStatus();
    </script>
</body>
</html>
