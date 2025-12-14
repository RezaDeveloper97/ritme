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
        .json-container {
            font-family: 'JetBrains Mono', 'Fira Code', 'Courier New', monospace;
            font-size: 12px;
            direction: ltr;
            text-align: left;
            line-height: 1.5;
        }
        .json-key { color: #e06c75; font-weight: 600; }
        .json-string { color: #98c379; }
        .json-number { color: #d19a66; }
        .json-boolean { color: #56b6c2; font-weight: 600; }
        .json-null { color: #c678dd; font-style: italic; }
        .json-bracket { color: #abb2bf; }
        .tab-active {
            border-bottom: 3px solid #ec4899;
            background: #fdf2f8;
        }
        .alert-emergency { background: #fee2e2; border-color: #dc2626; }
        .alert-warning { background: #fef3c7; border-color: #f59e0b; }
        .alert-info { background: #dbeafe; border-color: #3b82f6; }
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
                        <div class="bg-gray-900 rounded-xl p-4 min-h-[400px] max-h-[600px] overflow-auto">
                            <div id="profileResponse" class="json-container text-gray-100">
                                <span class="text-gray-500">// پاسخ API اینجا نمایش داده می‌شود</span>
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
                        <div class="bg-gray-900 rounded-xl p-4 min-h-[400px] max-h-[600px] overflow-auto">
                            <div id="symptomResponse" class="json-container text-gray-100">
                                <span class="text-gray-500">// پاسخ API اینجا نمایش داده می‌شود</span>
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
                        <div class="bg-gray-900 rounded-xl p-4 min-h-[400px] max-h-[600px] overflow-auto">
                            <div id="weeklyResponse" class="json-container text-gray-100">
                                <span class="text-gray-500">// پاسخ API اینجا نمایش داده می‌شود</span>
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
                        <div class="bg-gray-900 rounded-xl p-4 min-h-[400px] max-h-[600px] overflow-auto">
                            <div id="fetalResponse" class="json-container text-gray-100">
                                <span class="text-gray-500">// پاسخ API اینجا نمایش داده می‌شود</span>
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
                        <div class="bg-gray-900 rounded-xl p-4 min-h-[400px] max-h-[600px] overflow-auto">
                            <div id="alertsResponse" class="json-container text-gray-100">
                                <span class="text-gray-500">// پاسخ API اینجا نمایش داده می‌شود</span>
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
                        <div class="bg-gray-900 rounded-xl p-4 min-h-[400px] max-h-[600px] overflow-auto">
                            <div id="contentResponse" class="json-container text-gray-100">
                                <span class="text-gray-500">// پاسخ API اینجا نمایش داده می‌شود</span>
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

        // JSON formatter
        function formatJson(obj, indent = 0) {
            const pad = '  '.repeat(indent);
            const padInner = '  '.repeat(indent + 1);

            if (obj === null) return `<span class="json-null">null</span>`;
            if (typeof obj === 'boolean') return `<span class="json-boolean">${obj}</span>`;
            if (typeof obj === 'number') return `<span class="json-number">${obj}</span>`;
            if (typeof obj === 'string') {
                const escaped = obj.replace(/"/g, '\\"').replace(/\n/g, '\\n');
                return `<span class="json-string">"${escapeHtml(escaped)}"</span>`;
            }

            if (Array.isArray(obj)) {
                if (obj.length === 0) return '<span class="json-bracket">[]</span>';
                const items = obj.map(item => `${padInner}${formatJson(item, indent + 1)}`);
                return `<span class="json-bracket">[</span>\n${items.join(',\n')}\n${pad}<span class="json-bracket">]</span>`;
            }

            if (typeof obj === 'object') {
                const keys = Object.keys(obj);
                if (keys.length === 0) return '<span class="json-bracket">{}</span>';
                const items = keys.map(key => {
                    const val = formatJson(obj[key], indent + 1);
                    return `${padInner}<span class="json-key">"${escapeHtml(key)}"</span>: ${val}`;
                });
                return `<span class="json-bracket">{</span>\n${items.join(',\n')}\n${pad}<span class="json-bracket">}</span>`;
            }
            return String(obj);
        }

        function escapeHtml(str) {
            return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
        }

        function displayResponse(elementId, data) {
            document.getElementById(elementId).innerHTML = formatJson(data);
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
