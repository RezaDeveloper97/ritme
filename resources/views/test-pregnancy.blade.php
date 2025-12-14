<!DOCTYPE html>
<html lang="fa" dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="csrf-token" content="{{ csrf_token() }}">
    <title>صفحه تست - بارداری</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://cdn.jsdelivr.net/gh/AliR4M/vazirmatn/Vazirmatn-font-face.css" rel="stylesheet" type="text/css" />
    <style>
        body {
            font-family: 'Vazirmatn', sans-serif;
        }
        .tab-active {
            border-bottom: 3px solid #ec4899;
            background: #fdf2f8;
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
            border-right: 4px solid #ec4899;
        }
        .collapsible {
            cursor: pointer;
            user-select: none;
        }
        .collapsible:hover {
            background: #f9fafb;
        }
        .collapse-icon {
            transition: transform 0.2s;
        }
        .collapsed .collapse-icon {
            transform: rotate(-90deg);
        }
        .collapse-content {
            max-height: 2000px;
            overflow: hidden;
            transition: max-height 0.3s ease-out;
        }
        .collapsed + .collapse-content {
            max-height: 0;
        }
        .status-indicator {
            width: 10px;
            height: 10px;
            border-radius: 50%;
            display: inline-block;
            margin-left: 8px;
        }
        .status-active { background: #22c55e; }
        .status-inactive { background: #ef4444; }
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
            background: #ec4899;
            border-color: #ec4899;
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
    </style>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto px-4 py-6 max-w-7xl">
        <!-- Header -->
        <div class="bg-gradient-to-r from-pink-500 to-rose-600 rounded-2xl p-6 mb-6 text-white shadow-lg">
            <h1 class="text-3xl font-bold mb-2">صفحه تست بارداری</h1>
            <p class="opacity-90">تست API های Pregnancy Mode</p>
            <div class="mt-4 flex flex-wrap gap-4">
                <div class="bg-white/20 rounded-lg p-3">
                    <span class="text-sm">شماره تست: </span>
                    <span class="font-mono font-bold">09123456789</span>
                </div>
                <a href="/test-page" class="bg-white/20 hover:bg-white/30 rounded-lg p-3 transition">
                    <span class="text-sm">برگشت به صفحه تست اصلی</span>
                </a>
            </div>
        </div>

        <!-- Tabs -->
        <div class="bg-white rounded-t-xl shadow-lg overflow-hidden">
            <div class="flex border-b overflow-x-auto">
                <button onclick="switchTab('profile')" id="tab-profile" class="px-6 py-4 font-bold text-gray-700 hover:bg-gray-50 transition whitespace-nowrap tab-active">
                    پروفایل بارداری
                </button>
                <button onclick="switchTab('symptoms')" id="tab-symptoms" class="px-6 py-4 font-bold text-gray-700 hover:bg-gray-50 transition whitespace-nowrap">
                    علائم روزانه
                </button>
                <button onclick="switchTab('weekly')" id="tab-weekly" class="px-6 py-4 font-bold text-gray-700 hover:bg-gray-50 transition whitespace-nowrap">
                    گزارش هفتگی
                </button>
                <button onclick="switchTab('fetal')" id="tab-fetal" class="px-6 py-4 font-bold text-gray-700 hover:bg-gray-50 transition whitespace-nowrap">
                    حرکات جنین
                </button>
                <button onclick="switchTab('alerts')" id="tab-alerts" class="px-6 py-4 font-bold text-gray-700 hover:bg-gray-50 transition whitespace-nowrap">
                    هشدارها
                </button>
                <button onclick="switchTab('content')" id="tab-content" class="px-6 py-4 font-bold text-gray-700 hover:bg-gray-50 transition whitespace-nowrap">
                    محتوای هفتگی
                </button>
            </div>
        </div>

        <!-- Tab Contents -->
        <div class="bg-white rounded-b-xl shadow-lg p-6">
            <!-- Profile Tab -->
            <div id="content-profile" class="tab-content">
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <!-- Profile Actions -->
                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">مدیریت حالت بارداری</h3>
                        <div class="space-y-3 mb-6">
                            <button onclick="activatePregnancy()" class="w-full bg-green-500 hover:bg-green-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                فعال‌سازی حالت بارداری
                            </button>
                            <button onclick="deactivatePregnancy()" class="w-full bg-red-500 hover:bg-red-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                غیرفعال‌سازی حالت بارداری
                            </button>
                            <button onclick="getProfile()" class="w-full bg-blue-500 hover:bg-blue-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                دریافت پروفایل
                            </button>
                            <button onclick="getStatus()" class="w-full bg-purple-500 hover:bg-purple-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                دریافت وضعیت
                            </button>
                        </div>

                        <h3 class="text-lg font-bold mb-4 text-gray-800">آنبوردینگ بارداری</h3>
                        <form id="onboardingForm" class="space-y-4">
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">منبع محاسبه سن بارداری</label>
                                <select name="age_source" id="age_source" class="w-full px-4 py-2 border border-gray-300 rounded-lg" onchange="updateAgeSourceFields()">
                                    <option value="">انتخاب کنید</option>
                                </select>
                            </div>

                            <div id="lmp_fields" class="hidden">
                                <label class="block text-sm font-medium text-gray-700 mb-1">تاریخ آخرین پریود (LMP)</label>
                                <input type="date" name="lmp_date" id="lmp_date" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                            </div>

                            <div id="ultrasound_fields" class="hidden space-y-3">
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 mb-1">تاریخ سونوگرافی</label>
                                    <input type="date" name="ultrasound_date" id="ultrasound_date" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                </div>
                                <div class="grid grid-cols-2 gap-3">
                                    <div>
                                        <label class="block text-sm font-medium text-gray-700 mb-1">هفته بارداری</label>
                                        <input type="number" name="ultrasound_weeks" min="1" max="42" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-700 mb-1">روز</label>
                                        <input type="number" name="ultrasound_days" min="0" max="6" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                    </div>
                                </div>
                            </div>

                            <div id="manual_fields" class="hidden">
                                <div class="grid grid-cols-2 gap-3">
                                    <div>
                                        <label class="block text-sm font-medium text-gray-700 mb-1">هفته بارداری فعلی</label>
                                        <input type="number" name="manual_weeks" min="1" max="42" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-700 mb-1">روز</label>
                                        <input type="number" name="manual_days" min="0" max="6" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                    </div>
                                </div>
                            </div>

                            <div class="grid grid-cols-2 gap-3">
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 mb-1">گروه خونی</label>
                                    <select name="blood_type" id="blood_type" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                        <option value="">انتخاب کنید</option>
                                    </select>
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 mb-1">RH</label>
                                    <select name="rh_factor" id="rh_factor" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                        <option value="">انتخاب کنید</option>
                                    </select>
                                </div>
                            </div>

                            <div class="space-y-2">
                                <label class="flex items-center">
                                    <input type="checkbox" name="has_miscarriage_history" class="ml-2">
                                    <span class="text-sm text-gray-700">سابقه سقط</span>
                                </label>
                                <label class="flex items-center">
                                    <input type="checkbox" name="has_high_risk_history" class="ml-2">
                                    <span class="text-sm text-gray-700">سابقه بارداری پرخطر</span>
                                </label>
                            </div>

                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">بیماری‌های زمینه‌ای</label>
                                <div id="conditions_container" class="flex flex-wrap gap-2"></div>
                            </div>

                            <button type="submit" class="w-full bg-pink-500 hover:bg-pink-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                ثبت آنبوردینگ
                            </button>
                        </form>
                    </div>

                    <!-- Response Display -->
                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">پاسخ API</h3>
                        <div class="view-toggle">
                            <button onclick="setViewMode('profileResponse', 'pretty')" class="active" data-view="pretty">نمایش خوانا</button>
                            <button onclick="setViewMode('profileResponse', 'raw')" data-view="raw">JSON خام</button>
                        </div>
                        <div class="response-container" id="profileResponse">
                            <div class="text-center text-gray-400 py-12">
                                <svg class="w-16 h-16 mx-auto mb-4 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                                </svg>
                                <p>پاسخ API اینجا نمایش داده می‌شود</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Symptoms Tab -->
            <div id="content-symptoms" class="tab-content hidden">
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">ثبت علائم روزانه</h3>
                        <form id="symptomForm" class="space-y-4">
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">تاریخ</label>
                                <input type="date" name="log_date" id="symptom_date" class="w-full px-4 py-2 border border-gray-300 rounded-lg" required>
                            </div>

                            <div class="grid grid-cols-2 gap-4">
                                <div class="space-y-3">
                                    <h4 class="font-medium text-gray-700">علائم عمومی</h4>
                                    <label class="flex items-center">
                                        <input type="checkbox" name="has_nausea" class="ml-2">
                                        <span class="text-sm">تهوع</span>
                                    </label>
                                    <label class="flex items-center">
                                        <input type="checkbox" name="has_vomiting" class="ml-2">
                                        <span class="text-sm">استفراغ</span>
                                    </label>
                                    <label class="flex items-center">
                                        <input type="checkbox" name="has_fatigue" class="ml-2">
                                        <span class="text-sm">خستگی</span>
                                    </label>
                                    <label class="flex items-center">
                                        <input type="checkbox" name="has_headache" class="ml-2">
                                        <span class="text-sm">سردرد</span>
                                    </label>
                                    <label class="flex items-center">
                                        <input type="checkbox" name="has_dizziness" class="ml-2">
                                        <span class="text-sm">سرگیجه</span>
                                    </label>
                                </div>
                                <div class="space-y-3">
                                    <h4 class="font-medium text-gray-700">علائم درد</h4>
                                    <label class="flex items-center">
                                        <input type="checkbox" name="has_breast_pain" class="ml-2">
                                        <span class="text-sm">درد سینه</span>
                                    </label>
                                    <label class="flex items-center">
                                        <input type="checkbox" name="has_lower_abdominal_pain" class="ml-2">
                                        <span class="text-sm">درد پایین شکم</span>
                                    </label>
                                    <label class="flex items-center">
                                        <input type="checkbox" name="has_cramping" class="ml-2">
                                        <span class="text-sm">کرامپ</span>
                                    </label>
                                    <label class="flex items-center">
                                        <input type="checkbox" name="has_back_pain" class="ml-2">
                                        <span class="text-sm">کمردرد</span>
                                    </label>
                                    <label class="flex items-center">
                                        <input type="checkbox" name="has_pelvic_pressure" class="ml-2">
                                        <span class="text-sm">فشار لگن</span>
                                    </label>
                                </div>
                            </div>

                            <div class="space-y-3">
                                <h4 class="font-medium text-gray-700 text-red-600">علائم هشداردهنده</h4>
                                <div class="grid grid-cols-2 gap-4">
                                    <label class="flex items-center">
                                        <input type="checkbox" name="has_spotting" class="ml-2">
                                        <span class="text-sm">لکه‌بینی</span>
                                    </label>
                                    <label class="flex items-center">
                                        <input type="checkbox" name="has_bleeding" class="ml-2">
                                        <span class="text-sm">خونریزی</span>
                                    </label>
                                    <label class="flex items-center">
                                        <input type="checkbox" name="has_fluid_leakage" class="ml-2">
                                        <span class="text-sm">نشت مایع</span>
                                    </label>
                                    <label class="flex items-center">
                                        <input type="checkbox" name="has_severe_sudden_pain" class="ml-2">
                                        <span class="text-sm">درد شدید ناگهانی</span>
                                    </label>
                                </div>
                            </div>

                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">یادداشت</label>
                                <textarea name="notes" rows="3" class="w-full px-4 py-2 border border-gray-300 rounded-lg"></textarea>
                            </div>

                            <button type="submit" class="w-full bg-pink-500 hover:bg-pink-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                ثبت علائم
                            </button>
                        </form>

                        <div class="mt-6 space-y-3">
                            <button onclick="getSymptoms()" class="w-full bg-blue-500 hover:bg-blue-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                دریافت لیست علائم
                            </button>
                            <div class="flex gap-3">
                                <input type="date" id="symptom_date_search" class="flex-1 px-4 py-2 border border-gray-300 rounded-lg">
                                <button onclick="getSymptomByDate()" class="bg-purple-500 hover:bg-purple-600 text-white font-bold py-2 px-4 rounded-lg transition">
                                    جستجو
                                </button>
                                <button onclick="deleteSymptom()" class="bg-red-500 hover:bg-red-600 text-white font-bold py-2 px-4 rounded-lg transition">
                                    حذف
                                </button>
                            </div>
                        </div>
                    </div>

                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">پاسخ API</h3>
                        <div class="view-toggle">
                            <button onclick="setViewMode('symptomResponse', 'pretty')" class="active" data-view="pretty">نمایش خوانا</button>
                            <button onclick="setViewMode('symptomResponse', 'raw')" data-view="raw">JSON خام</button>
                        </div>
                        <div class="response-container" id="symptomResponse">
                            <div class="text-center text-gray-400 py-12">
                                <p>پاسخ API اینجا نمایش داده می‌شود</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Weekly Tab -->
            <div id="content-weekly" class="tab-content hidden">
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">ثبت گزارش هفتگی</h3>
                        <form id="weeklyForm" class="space-y-4">
                            <div class="grid grid-cols-2 gap-3">
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 mb-1">تاریخ</label>
                                    <input type="date" name="log_date" class="w-full px-4 py-2 border border-gray-300 rounded-lg" required>
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 mb-1">هفته بارداری</label>
                                    <input type="number" name="pregnancy_week" min="1" max="42" class="w-full px-4 py-2 border border-gray-300 rounded-lg" required>
                                </div>
                            </div>

                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">وزن (کیلوگرم)</label>
                                <input type="number" name="weight" step="0.1" min="30" max="200" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                            </div>

                            <div class="space-y-3">
                                <label class="flex items-center">
                                    <input type="checkbox" name="has_swelling" class="ml-2">
                                    <span class="text-sm">ورم دارم</span>
                                </label>
                                <div id="swelling_locations" class="flex flex-wrap gap-2 mr-6"></div>
                            </div>

                            <label class="flex items-center">
                                <input type="checkbox" name="has_shortness_of_breath" class="ml-2">
                                <span class="text-sm">تنگی نفس دارم</span>
                            </label>

                            <div class="space-y-3">
                                <label class="flex items-center">
                                    <input type="checkbox" name="has_blood_pressure_device" id="bp_device" class="ml-2" onchange="toggleBPFields()">
                                    <span class="text-sm">دستگاه فشارسنج دارم</span>
                                </label>
                                <div id="bp_fields" class="hidden grid grid-cols-2 gap-3 mr-6">
                                    <div>
                                        <label class="block text-sm font-medium text-gray-700 mb-1">فشار سیستولیک</label>
                                        <input type="number" name="systolic_pressure" min="60" max="250" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                    </div>
                                    <div>
                                        <label class="block text-sm font-medium text-gray-700 mb-1">فشار دیاستولیک</label>
                                        <input type="number" name="diastolic_pressure" min="40" max="150" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                    </div>
                                </div>
                            </div>

                            <div class="grid grid-cols-2 gap-3">
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 mb-1">قند خون ناشتا</label>
                                    <input type="number" name="fasting_blood_sugar" step="0.1" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 mb-1">قند بعد غذا</label>
                                    <input type="number" name="post_meal_blood_sugar" step="0.1" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                </div>
                            </div>

                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">حال کلی</label>
                                <select name="overall_mood" id="overall_mood" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                    <option value="">انتخاب کنید</option>
                                </select>
                            </div>

                            <div class="space-y-3">
                                <label class="flex items-center">
                                    <input type="checkbox" name="has_anxiety" class="ml-2">
                                    <span class="text-sm">اضطراب دارم</span>
                                </label>
                                <label class="flex items-center">
                                    <input type="checkbox" name="has_mood_swings" class="ml-2">
                                    <span class="text-sm">نوسان خلقی دارم</span>
                                </label>
                                <label class="flex items-center">
                                    <input type="checkbox" name="has_depression_feelings" class="ml-2">
                                    <span class="text-sm">احساس افسردگی دارم</span>
                                </label>
                            </div>

                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">یادداشت</label>
                                <textarea name="notes" rows="2" class="w-full px-4 py-2 border border-gray-300 rounded-lg"></textarea>
                            </div>

                            <button type="submit" class="w-full bg-pink-500 hover:bg-pink-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                ثبت گزارش هفتگی
                            </button>
                        </form>

                        <div class="mt-6 space-y-3">
                            <button onclick="getWeeklyLogs()" class="w-full bg-blue-500 hover:bg-blue-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                دریافت لیست گزارشات هفتگی
                            </button>
                            <div class="flex gap-3">
                                <input type="number" id="weekly_week_search" min="1" max="42" placeholder="شماره هفته" class="flex-1 px-4 py-2 border border-gray-300 rounded-lg">
                                <button onclick="getWeeklyLog()" class="bg-purple-500 hover:bg-purple-600 text-white font-bold py-2 px-4 rounded-lg transition">
                                    جستجو
                                </button>
                            </div>
                        </div>
                    </div>

                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">پاسخ API</h3>
                        <div class="view-toggle">
                            <button onclick="setViewMode('weeklyResponse', 'pretty')" class="active" data-view="pretty">نمایش خوانا</button>
                            <button onclick="setViewMode('weeklyResponse', 'raw')" data-view="raw">JSON خام</button>
                        </div>
                        <div class="response-container" id="weeklyResponse">
                            <div class="text-center text-gray-400 py-12">
                                <p>پاسخ API اینجا نمایش داده می‌شود</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Fetal Movement Tab -->
            <div id="content-fetal" class="tab-content hidden">
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">ثبت حرکات جنین</h3>
                        <form id="fetalForm" class="space-y-4">
                            <div class="grid grid-cols-2 gap-3">
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 mb-1">تاریخ</label>
                                    <input type="date" name="log_date" class="w-full px-4 py-2 border border-gray-300 rounded-lg" required>
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 mb-1">هفته بارداری</label>
                                    <input type="number" name="pregnancy_week" min="1" max="42" class="w-full px-4 py-2 border border-gray-300 rounded-lg" required>
                                </div>
                            </div>

                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">وضعیت حرکات</label>
                                <select name="movement_status" id="movement_status" class="w-full px-4 py-2 border border-gray-300 rounded-lg" required>
                                    <option value="">انتخاب کنید</option>
                                </select>
                            </div>

                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">تعداد حرکات (در یک ساعت)</label>
                                <input type="number" name="movement_count" min="0" max="100" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                            </div>

                            <div class="grid grid-cols-2 gap-3">
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 mb-1">ساعت اولین حرکت</label>
                                    <input type="time" name="first_movement_time" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                </div>
                                <div>
                                    <label class="block text-sm font-medium text-gray-700 mb-1">ساعت آخرین حرکت</label>
                                    <input type="time" name="last_movement_time" class="w-full px-4 py-2 border border-gray-300 rounded-lg">
                                </div>
                            </div>

                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">یادداشت</label>
                                <textarea name="notes" rows="2" class="w-full px-4 py-2 border border-gray-300 rounded-lg"></textarea>
                            </div>

                            <button type="submit" class="w-full bg-pink-500 hover:bg-pink-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                ثبت حرکات جنین
                            </button>
                        </form>

                        <div class="mt-6">
                            <button onclick="getFetalMovements()" class="w-full bg-blue-500 hover:bg-blue-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                دریافت لیست حرکات جنین
                            </button>
                        </div>
                    </div>

                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">پاسخ API</h3>
                        <div class="view-toggle">
                            <button onclick="setViewMode('fetalResponse', 'pretty')" class="active" data-view="pretty">نمایش خوانا</button>
                            <button onclick="setViewMode('fetalResponse', 'raw')" data-view="raw">JSON خام</button>
                        </div>
                        <div class="response-container" id="fetalResponse">
                            <div class="text-center text-gray-400 py-12">
                                <p>پاسخ API اینجا نمایش داده می‌شود</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Alerts Tab -->
            <div id="content-alerts" class="tab-content hidden">
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">مدیریت هشدارها</h3>
                        <div class="space-y-3">
                            <button onclick="getAlerts()" class="w-full bg-blue-500 hover:bg-blue-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                دریافت لیست هشدارها
                            </button>
                            <button onclick="getAlertsSummary()" class="w-full bg-purple-500 hover:bg-purple-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                دریافت خلاصه هشدارها
                            </button>
                            <button onclick="markAllAlertsAsRead()" class="w-full bg-green-500 hover:bg-green-600 text-white font-bold py-3 px-6 rounded-lg transition">
                                خواندن همه هشدارها
                            </button>
                        </div>

                        <div class="mt-6 space-y-3">
                            <h4 class="font-medium text-gray-700">عملیات بر روی هشدار خاص</h4>
                            <div class="flex gap-3">
                                <input type="number" id="alert_id" placeholder="شناسه هشدار" class="flex-1 px-4 py-2 border border-gray-300 rounded-lg">
                                <button onclick="markAlertAsRead()" class="bg-green-500 hover:bg-green-600 text-white font-bold py-2 px-4 rounded-lg transition">
                                    خوانده شد
                                </button>
                                <button onclick="dismissAlert()" class="bg-red-500 hover:bg-red-600 text-white font-bold py-2 px-4 rounded-lg transition">
                                    رد
                                </button>
                            </div>
                        </div>

                        <div class="mt-6">
                            <h4 class="font-medium text-gray-700 mb-3">فیلتر هشدارها</h4>
                            <div class="space-y-3">
                                <select id="alert_level_filter" class="w-full px-4 py-2 border border-gray-300 rounded-lg" onchange="getAlerts()">
                                    <option value="">همه سطوح</option>
                                </select>
                                <label class="flex items-center">
                                    <input type="checkbox" id="unread_only" class="ml-2" onchange="getAlerts()">
                                    <span class="text-sm">فقط خوانده نشده‌ها</span>
                                </label>
                            </div>
                        </div>
                    </div>

                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">پاسخ API</h3>
                        <div class="view-toggle">
                            <button onclick="setViewMode('alertsResponse', 'pretty')" class="active" data-view="pretty">نمایش خوانا</button>
                            <button onclick="setViewMode('alertsResponse', 'raw')" data-view="raw">JSON خام</button>
                        </div>
                        <div class="response-container" id="alertsResponse">
                            <div class="text-center text-gray-400 py-12">
                                <p>پاسخ API اینجا نمایش داده می‌شود</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Content Tab -->
            <div id="content-content" class="tab-content hidden">
                <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">محتوای آموزشی هفتگی</h3>
                        <div class="flex gap-3 mb-6">
                            <input type="number" id="content_week" min="1" max="40" placeholder="شماره هفته (1-40)" class="flex-1 px-4 py-2 border border-gray-300 rounded-lg">
                            <button onclick="getWeeklyContent()" class="bg-pink-500 hover:bg-pink-600 text-white font-bold py-2 px-6 rounded-lg transition">
                                دریافت محتوا
                            </button>
                        </div>

                        <div class="grid grid-cols-5 gap-2">
                            @for($i = 1; $i <= 40; $i++)
                                <button onclick="document.getElementById('content_week').value = {{ $i }}; getWeeklyContent();"
                                    class="py-2 px-3 bg-gray-100 hover:bg-pink-100 rounded-lg text-sm font-medium transition">
                                    {{ $i }}
                                </button>
                            @endfor
                        </div>
                    </div>

                    <div>
                        <h3 class="text-lg font-bold mb-4 text-gray-800">پاسخ API</h3>
                        <div class="view-toggle">
                            <button onclick="setViewMode('contentResponse', 'pretty')" class="active" data-view="pretty">نمایش خوانا</button>
                            <button onclick="setViewMode('contentResponse', 'raw')" data-view="raw">JSON خام</button>
                        </div>
                        <div class="response-container" id="contentResponse">
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
        let responseData = {}; // Store raw data for each response element
        let viewModes = {}; // Store view mode for each element

        // Persian translations for keys
        const keyTranslations = {
            // Common
            'success': 'موفقیت',
            'message': 'پیام',
            'data': 'اطلاعات',
            'id': 'شناسه',
            'user_id': 'شناسه کاربر',
            'created_at': 'تاریخ ایجاد',
            'updated_at': 'تاریخ بروزرسانی',
            'notes': 'یادداشت',
            'count': 'تعداد',
            'logs': 'گزارش‌ها',
            'alerts': 'هشدارها',
            'counts': 'آمار',

            // Profile
            'profile': 'پروفایل',
            'status': 'وضعیت',
            'pregnancy_mode': 'حالت بارداری',
            'cycle_mode': 'حالت سایکل',
            'onboarding_required': 'نیاز به آنبوردینگ',
            'onboarding_completed': 'آنبوردینگ تکمیل شده',
            'onboarding_completed_at': 'تاریخ تکمیل آنبوردینگ',
            'age_source': 'منبع محاسبه سن',
            'lmp_date': 'تاریخ آخرین پریود',
            'ultrasound_date': 'تاریخ سونوگرافی',
            'ultrasound_weeks': 'هفته سونوگرافی',
            'ultrasound_days': 'روز سونوگرافی',
            'manual_weeks': 'هفته دستی',
            'manual_days': 'روز دستی',
            'manual_entry_date': 'تاریخ ورود دستی',
            'confidence_level': 'سطح اطمینان',
            'uncertainty_days': 'روزهای عدم قطعیت',
            'estimated_due_date': 'تاریخ تخمینی زایمان',
            'blood_type': 'گروه خونی',
            'rh_factor': 'فاکتور RH',
            'rh_negative_care_flag': 'پرچم مراقبت RH منفی',
            'has_miscarriage_history': 'سابقه سقط',
            'has_high_risk_history': 'سابقه پرخطر',
            'pre_existing_conditions': 'بیماری‌های زمینه‌ای',
            'fetal_movement_felt': 'حس حرکت جنین',
            'first_fetal_movement_date': 'تاریخ اولین حرکت',
            'is_locked': 'قفل شده',

            // Status
            'is_active': 'فعال است',
            'gestational_age': 'سن بارداری',
            'weeks': 'هفته',
            'days': 'روز',
            'trimester': 'سه‌ماهه',
            'current_week': 'هفته فعلی',
            'is_high_risk': 'پرخطر است',
            'fetal_movement_tracking_active': 'ردیابی حرکت فعال',
            'fetal_movement_required': 'حرکت جنین الزامی',
            'flags': 'پرچم‌ها',
            'date': 'تاریخ',
            'range': 'بازه',
            'range_start': 'شروع بازه',
            'range_end': 'پایان بازه',

            // Symptoms
            'log': 'گزارش',
            'log_date': 'تاریخ گزارش',
            'pregnancy_week': 'هفته بارداری',
            'has_nausea': 'تهوع',
            'nausea_severity': 'شدت تهوع',
            'has_vomiting': 'استفراغ',
            'vomiting_severity': 'شدت استفراغ',
            'has_fatigue': 'خستگی',
            'fatigue_severity': 'شدت خستگی',
            'has_headache': 'سردرد',
            'headache_severity': 'شدت سردرد',
            'has_dizziness': 'سرگیجه',
            'dizziness_severity': 'شدت سرگیجه',
            'has_breast_pain': 'درد سینه',
            'breast_pain_severity': 'شدت درد سینه',
            'has_lower_abdominal_pain': 'درد زیر شکم',
            'lower_abdominal_pain_severity': 'شدت درد زیر شکم',
            'has_cramping': 'کرامپ',
            'cramping_severity': 'شدت کرامپ',
            'has_back_pain': 'کمردرد',
            'back_pain_severity': 'شدت کمردرد',
            'has_pelvic_pressure': 'فشار لگن',
            'pelvic_pressure_severity': 'شدت فشار لگن',
            'has_spotting': 'لکه‌بینی',
            'spotting_severity': 'شدت لکه‌بینی',
            'has_bleeding': 'خونریزی',
            'bleeding_severity': 'شدت خونریزی',
            'has_fluid_leakage': 'نشت مایع',
            'fluid_leakage_severity': 'شدت نشت مایع',
            'has_severe_sudden_pain': 'درد شدید ناگهانی',
            'severe_sudden_pain_severity': 'شدت درد ناگهانی',

            // Weekly
            'weight': 'وزن',
            'has_swelling': 'ورم',
            'swelling_locations': 'محل ورم',
            'has_shortness_of_breath': 'تنگی نفس',
            'has_blood_pressure_device': 'دستگاه فشارسنج',
            'systolic_pressure': 'فشار سیستولیک',
            'diastolic_pressure': 'فشار دیاستولیک',
            'fasting_blood_sugar': 'قند ناشتا',
            'post_meal_blood_sugar': 'قند بعد غذا',
            'overall_mood': 'حال کلی',
            'has_anxiety': 'اضطراب',
            'anxiety_severity': 'شدت اضطراب',
            'has_mood_swings': 'نوسان خلقی',
            'mood_swings_severity': 'شدت نوسان خلقی',
            'has_depression_feelings': 'احساس افسردگی',
            'depression_severity': 'شدت افسردگی',

            // Fetal Movement
            'movement_status': 'وضعیت حرکات',
            'movement_count': 'تعداد حرکات',
            'first_movement_time': 'زمان اولین حرکت',
            'last_movement_time': 'زمان آخرین حرکت',

            // Alerts
            'total': 'کل',
            'unread': 'خوانده نشده',
            'emergency': 'اورژانسی',
            'warning': 'هشدار',
            'info': 'اطلاعاتی',
            'has_emergency': 'هشدار اورژانسی دارد',
            'has_unread': 'خوانده نشده دارد',
            'latest_emergency': 'آخرین اورژانس',
            'level': 'سطح',
            'alert_type': 'نوع هشدار',
            'title': 'عنوان',
            'description': 'توضیحات',
            'action_required': 'اقدام لازم',
            'is_read': 'خوانده شده',
            'read_at': 'زمان خواندن',
            'is_dismissed': 'رد شده',
            'dismissed_at': 'زمان رد',
            'is_active': 'فعال',
            'expires_at': 'انقضا',

            // Content
            'content': 'محتوا',
            'week': 'هفته',
            'week_number': 'شماره هفته',
            'fetal_development': 'رشد جنین',
            'mother_body_changes': 'تغییرات بدن مادر',
            'dos_and_donts': 'بایدها و نبایدها',
            'care_plan': 'برنامه مراقبت',
            'body_adaptation': 'سازگاری بدن',
            'emotional_status': 'وضعیت احساسی',
            'key_nutrition': 'تغذیه کلیدی',
            'physical_activity': 'فعالیت بدنی',
            'tests_and_checkups': 'آزمایشات و معاینات',
            'faq': 'سوالات متداول',
        };

        // Value translations
        const valueTranslations = {
            'true': 'بله',
            'false': 'خیر',
            'null': 'ندارد',
            'lmp': 'آخرین پریود (LMP)',
            'ultrasound': 'سونوگرافی',
            'manual': 'ورود دستی',
            'high': 'بالا',
            'medium': 'متوسط',
            'low': 'پایین',
            'mild': 'خفیف',
            'moderate': 'متوسط',
            'severe': 'شدید',
            'good': 'خوب',
            'poor': 'بد',
            'positive': 'مثبت',
            'negative': 'منفی',
            'feet': 'پا',
            'hands': 'دست',
            'face': 'صورت',
            'not_felt_yet': 'هنوز حس نشده',
            'felt': 'حس شده',
            'normal': 'عادی',
            'reduced': 'کاهش یافته',
            'increased': 'افزایش یافته',
            'none': 'هیچ',
            'chronic_hypertension': 'فشار خون مزمن',
            'diabetes': 'دیابت',
            'hypothyroidism': 'کم‌کاری تیروئید',
            'hyperthyroidism': 'پرکاری تیروئید',
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

        function isDateString(str) {
            return /^\d{4}-\d{2}-\d{2}/.test(str);
        }

        function formatDate(dateStr) {
            if (!dateStr) return 'ندارد';
            try {
                const d = new Date(dateStr);
                return d.toLocaleDateString('fa-IR', { year: 'numeric', month: 'long', day: 'numeric' });
            } catch {
                return dateStr;
            }
        }

        // Pretty format for display
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
            } else if (typeof data === 'object' && !data.success) {
                html += renderObject(data, '');
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

            // Special handling for known sections
            const sections = {
                profile: 'پروفایل',
                status: 'وضعیت',
                gestational_age: 'سن بارداری',
                estimated_due_date: 'تاریخ تخمینی زایمان',
                log: 'گزارش',
                logs: 'گزارش‌ها',
                alerts: 'هشدارها',
                counts: 'آمار',
                content: 'محتوا',
                flags: 'پرچم‌ها',
            };

            for (const [key, label] of Object.entries(sections)) {
                if (obj[key] !== undefined) {
                    html += `<div class="data-card">`;
                    html += `<div class="data-card-header info">${label}</div>`;
                    html += `<div class="data-card-content">`;
                    if (Array.isArray(obj[key])) {
                        html += renderObject(obj[key], label);
                    } else if (typeof obj[key] === 'object' && obj[key] !== null) {
                        html += renderObjectRows(obj[key]);
                    } else {
                        html += `<span class="data-value">${translateValue(obj[key])}</span>`;
                    }
                    html += `</div></div>`;
                }
            }

            // Render remaining fields
            const renderedKeys = Object.keys(sections);
            const remainingKeys = Object.keys(obj).filter(k => !renderedKeys.includes(k));

            if (remainingKeys.length > 0) {
                const remainingObj = {};
                remainingKeys.forEach(k => remainingObj[k] = obj[k]);

                if (html === '') {
                    // No sections found, render as single card
                    html += `<div class="data-card">`;
                    if (title) html += `<div class="data-card-header info">${title}</div>`;
                    html += `<div class="data-card-content">`;
                    html += renderObjectRows(remainingObj);
                    html += `</div></div>`;
                } else {
                    // Add remaining to separate card
                    html += `<div class="data-card">`;
                    html += `<div class="data-card-header info">سایر اطلاعات</div>`;
                    html += `<div class="data-card-content">`;
                    html += renderObjectRows(remainingObj);
                    html += `</div></div>`;
                }
            }

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
                            html += `<span class="badge badge-purple ml-1">${translateValue(v)}</span>`;
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
                } else if (isDateString(String(value))) {
                    html += `<div class="data-row">
                        <span class="data-label">${label}</span>
                        <span class="data-value date">${formatDate(value)}</span>
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
                container.innerHTML = formatPretty(data, elementId);
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
            const result = await apiCall('/test-pregnancy/enums');
            if (result.success) {
                enums = result.data;
                populateEnumSelects();
            }
        }

        function populateEnumSelects() {
            // Age sources
            const ageSourceSelect = document.getElementById('age_source');
            enums.age_sources?.forEach(item => {
                const option = document.createElement('option');
                option.value = item.value;
                option.textContent = item.label;
                ageSourceSelect.appendChild(option);
            });

            // Blood types
            const bloodTypeSelect = document.getElementById('blood_type');
            enums.blood_types?.forEach(item => {
                const option = document.createElement('option');
                option.value = item.value;
                option.textContent = item.label;
                bloodTypeSelect.appendChild(option);
            });

            // RH factors
            const rhSelect = document.getElementById('rh_factor');
            enums.rh_factors?.forEach(item => {
                const option = document.createElement('option');
                option.value = item.value;
                option.textContent = item.label;
                rhSelect.appendChild(option);
            });

            // Pre-existing conditions
            const conditionsContainer = document.getElementById('conditions_container');
            enums.pre_existing_conditions?.forEach(item => {
                const label = document.createElement('label');
                label.className = 'inline-flex items-center px-3 py-1 bg-gray-100 rounded-lg cursor-pointer hover:bg-gray-200';
                label.innerHTML = `<input type="checkbox" name="pre_existing_conditions[]" value="${item.value}" class="ml-2"><span class="text-sm">${item.label}</span>`;
                conditionsContainer.appendChild(label);
            });

            // Overall mood
            const moodSelect = document.getElementById('overall_mood');
            enums.overall_mood?.forEach(item => {
                const option = document.createElement('option');
                option.value = item.value;
                option.textContent = item.label;
                moodSelect.appendChild(option);
            });

            // Swelling locations
            const swellingContainer = document.getElementById('swelling_locations');
            enums.swelling_locations?.forEach(item => {
                const label = document.createElement('label');
                label.className = 'inline-flex items-center px-3 py-1 bg-gray-100 rounded-lg cursor-pointer hover:bg-gray-200';
                label.innerHTML = `<input type="checkbox" name="swelling_locations[]" value="${item.value}" class="ml-2"><span class="text-sm">${item.label}</span>`;
                swellingContainer.appendChild(label);
            });

            // Fetal movement status
            const movementSelect = document.getElementById('movement_status');
            enums.fetal_movement_status?.forEach(item => {
                const option = document.createElement('option');
                option.value = item.value;
                option.textContent = item.label;
                movementSelect.appendChild(option);
            });

            // Alert levels filter
            const alertLevelSelect = document.getElementById('alert_level_filter');
            enums.alert_levels?.forEach(item => {
                const option = document.createElement('option');
                option.value = item.value;
                option.textContent = item.label;
                alertLevelSelect.appendChild(option);
            });
        }

        function updateAgeSourceFields() {
            const source = document.getElementById('age_source').value;
            document.getElementById('lmp_fields').classList.add('hidden');
            document.getElementById('ultrasound_fields').classList.add('hidden');
            document.getElementById('manual_fields').classList.add('hidden');

            if (source === 'lmp') {
                document.getElementById('lmp_fields').classList.remove('hidden');
            } else if (source === 'ultrasound') {
                document.getElementById('ultrasound_fields').classList.remove('hidden');
            } else if (source === 'manual') {
                document.getElementById('manual_fields').classList.remove('hidden');
            }
        }

        function toggleBPFields() {
            const checked = document.getElementById('bp_device').checked;
            document.getElementById('bp_fields').classList.toggle('hidden', !checked);
        }

        // Profile functions
        async function activatePregnancy() {
            const result = await apiCall('/test-pregnancy/activate', 'POST');
            displayResponse('profileResponse', result);
        }

        async function deactivatePregnancy() {
            const result = await apiCall('/test-pregnancy/deactivate', 'POST');
            displayResponse('profileResponse', result);
        }

        async function getProfile() {
            const result = await apiCall('/test-pregnancy/profile');
            displayResponse('profileResponse', result);
        }

        async function getStatus() {
            const result = await apiCall('/test-pregnancy/status');
            displayResponse('profileResponse', result);
        }

        // Onboarding form
        document.getElementById('onboardingForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const data = {};

            for (const [key, value] of formData.entries()) {
                if (key === 'pre_existing_conditions[]') {
                    if (!data.pre_existing_conditions) data.pre_existing_conditions = [];
                    data.pre_existing_conditions.push(value);
                } else if (key.startsWith('has_')) {
                    data[key] = true;
                } else if (value) {
                    data[key] = value;
                }
            }

            const result = await apiCall('/test-pregnancy/onboarding', 'POST', data);
            displayResponse('profileResponse', result);
        });

        // Symptom functions
        async function getSymptoms() {
            const result = await apiCall('/test-pregnancy/symptoms');
            displayResponse('symptomResponse', result);
        }

        async function getSymptomByDate() {
            const date = document.getElementById('symptom_date_search').value;
            if (!date) {
                alert('لطفا تاریخ را انتخاب کنید');
                return;
            }
            const result = await apiCall(`/test-pregnancy/symptoms/${date}`);
            displayResponse('symptomResponse', result);
        }

        async function deleteSymptom() {
            const date = document.getElementById('symptom_date_search').value;
            if (!date) {
                alert('لطفا تاریخ را انتخاب کنید');
                return;
            }
            if (!confirm('آیا از حذف این رکورد اطمینان دارید؟')) return;
            const result = await apiCall(`/test-pregnancy/symptoms/${date}`, 'DELETE');
            displayResponse('symptomResponse', result);
        }

        document.getElementById('symptomForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const data = {};

            for (const [key, value] of formData.entries()) {
                if (key.startsWith('has_')) {
                    data[key] = true;
                } else if (value) {
                    data[key] = value;
                }
            }

            const result = await apiCall('/test-pregnancy/symptoms', 'POST', data);
            displayResponse('symptomResponse', result);
        });

        // Weekly functions
        async function getWeeklyLogs() {
            const result = await apiCall('/test-pregnancy/weekly');
            displayResponse('weeklyResponse', result);
        }

        async function getWeeklyLog() {
            const week = document.getElementById('weekly_week_search').value;
            if (!week) {
                alert('لطفا شماره هفته را وارد کنید');
                return;
            }
            const result = await apiCall(`/test-pregnancy/weekly/${week}`);
            displayResponse('weeklyResponse', result);
        }

        document.getElementById('weeklyForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const data = {};

            for (const [key, value] of formData.entries()) {
                if (key === 'swelling_locations[]') {
                    if (!data.swelling_locations) data.swelling_locations = [];
                    data.swelling_locations.push(value);
                } else if (key.startsWith('has_')) {
                    data[key] = true;
                } else if (value) {
                    data[key] = value;
                }
            }

            const result = await apiCall('/test-pregnancy/weekly', 'POST', data);
            displayResponse('weeklyResponse', result);
        });

        // Fetal movement functions
        async function getFetalMovements() {
            const result = await apiCall('/test-pregnancy/fetal-movement');
            displayResponse('fetalResponse', result);
        }

        document.getElementById('fetalForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const data = {};

            for (const [key, value] of formData.entries()) {
                if (value) {
                    data[key] = value;
                }
            }

            const result = await apiCall('/test-pregnancy/fetal-movement', 'POST', data);
            displayResponse('fetalResponse', result);
        });

        // Alert functions
        async function getAlerts() {
            const level = document.getElementById('alert_level_filter').value;
            const unreadOnly = document.getElementById('unread_only').checked;
            let url = '/test-pregnancy/alerts';
            const params = [];
            if (level) params.push(`level=${level}`);
            if (unreadOnly) params.push('unread_only=1');
            if (params.length) url += '?' + params.join('&');
            const result = await apiCall(url);
            displayResponse('alertsResponse', result);
        }

        async function getAlertsSummary() {
            const result = await apiCall('/test-pregnancy/alerts/summary');
            displayResponse('alertsResponse', result);
        }

        async function markAlertAsRead() {
            const id = document.getElementById('alert_id').value;
            if (!id) {
                alert('لطفا شناسه هشدار را وارد کنید');
                return;
            }
            const result = await apiCall(`/test-pregnancy/alerts/${id}/read`, 'POST');
            displayResponse('alertsResponse', result);
        }

        async function markAllAlertsAsRead() {
            const result = await apiCall('/test-pregnancy/alerts/read-all', 'POST');
            displayResponse('alertsResponse', result);
        }

        async function dismissAlert() {
            const id = document.getElementById('alert_id').value;
            if (!id) {
                alert('لطفا شناسه هشدار را وارد کنید');
                return;
            }
            const result = await apiCall(`/test-pregnancy/alerts/${id}/dismiss`, 'POST');
            displayResponse('alertsResponse', result);
        }

        // Content function
        async function getWeeklyContent() {
            const week = document.getElementById('content_week').value;
            if (!week) {
                alert('لطفا شماره هفته را وارد کنید');
                return;
            }
            const result = await apiCall(`/test-pregnancy/content/${week}`);
            displayResponse('contentResponse', result);
        }

        // Set default dates
        function setDefaultDates() {
            const today = new Date().toISOString().split('T')[0];
            document.getElementById('symptom_date').value = today;
            document.getElementById('symptom_date_search').value = today;
            document.querySelectorAll('input[type="date"]').forEach(el => {
                if (!el.value) el.value = today;
            });
        }

        // Initialize
        loadEnums();
        setDefaultDates();
    </script>
</body>
</html>
