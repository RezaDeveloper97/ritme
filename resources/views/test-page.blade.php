<!DOCTYPE html>
<html lang="fa" dir="rtl">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="csrf-token" content="{{ csrf_token() }}">
    <title>صفحه تست - پروفایل و سایکل</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <link href="https://cdn.jsdelivr.net/gh/AliR4M/vazirmatn/Vazirmatn-font-face.css" rel="stylesheet" type="text/css" />
    <style>
        body {
            font-family: 'Vazirmatn', sans-serif;
        }
        .json-container {
            font-family: 'JetBrains Mono', 'Fira Code', 'Courier New', monospace;
            font-size: 13px;
            direction: ltr;
            text-align: left;
            line-height: 1.6;
        }
        .json-key { color: #e06c75; font-weight: 600; }
        .json-string { color: #98c379; }
        .json-number { color: #d19a66; }
        .json-boolean { color: #56b6c2; font-weight: 600; }
        .json-null { color: #c678dd; font-style: italic; }
        .json-bracket { color: #abb2bf; }
        .json-row {
            padding: 4px 12px;
            border-radius: 4px;
            margin: 2px 0;
        }
        .json-row:hover {
            background: rgba(255,255,255,0.05);
        }
        .phase-badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: bold;
        }
    </style>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto px-4 py-8 max-w-6xl">
        <!-- Header -->
        <div class="bg-gradient-to-r from-pink-500 to-purple-600 rounded-2xl p-6 mb-8 text-white shadow-lg">
            <h1 class="text-3xl font-bold mb-2">صفحه تست</h1>
            <p class="opacity-90">تست پروفایل و محاسبات سایکل</p>
            <div class="mt-4 bg-white/20 rounded-lg p-3 inline-block">
                <span class="text-sm">شماره تست: </span>
                <span class="font-mono font-bold">09123456789</span>
            </div>
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
            <!-- Profile Section -->
            <div class="bg-white rounded-2xl shadow-lg overflow-hidden">
                <div class="bg-purple-500 text-white px-6 py-4">
                    <h2 class="text-xl font-bold">پروفایل کاربر</h2>
                </div>
                <div class="p-6">
                    <form id="profileForm" class="space-y-4">
                        <div>
                            <label class="block text-sm font-medium text-gray-700 mb-1">نام</label>
                            <input type="text" name="name" id="name" value="{{ $user->name ?? '' }}"
                                class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent">
                        </div>

                        <div>
                            <label class="block text-sm font-medium text-gray-700 mb-1">تاریخ تولد</label>
                            <input type="date" name="birthday" id="birthday" value="{{ $profile->birthday ?? '' }}"
                                class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent">
                        </div>

                        <div class="grid grid-cols-2 gap-4">
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">وزن (کیلوگرم)</label>
                                <input type="number" name="weight" id="weight" value="{{ $profile->weight ?? '' }}" step="0.1"
                                    class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent">
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">قد (سانتی‌متر)</label>
                                <input type="number" name="height" id="height" value="{{ $profile->height ?? '' }}"
                                    class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent">
                            </div>
                        </div>

                        <div class="grid grid-cols-2 gap-4">
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">طول پریود (روز)</label>
                                <input type="number" name="period_duration" id="period_duration" value="{{ $profile->period_duration ?? '' }}" min="1" max="15"
                                    class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent">
                            </div>
                            <div>
                                <label class="block text-sm font-medium text-gray-700 mb-1">طول سایکل (روز)</label>
                                <input type="number" name="cycle_duration" id="cycle_duration" value="{{ $profile->cycle_duration ?? '' }}" min="15" max="60"
                                    class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent">
                            </div>
                        </div>

                        <div>
                            <label class="block text-sm font-medium text-gray-700 mb-1">تاریخ شروع آخرین پریود</label>
                            <input type="date" name="last_period_start" id="last_period_start" value="{{ $profile->last_period_start ?? '' }}"
                                class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent">
                        </div>

                        <button type="submit" id="saveProfileBtn"
                            class="w-full bg-purple-500 hover:bg-purple-600 text-white font-bold py-3 px-6 rounded-lg transition duration-200">
                            ذخیره پروفایل
                        </button>
                    </form>

                    <div id="profileStatus" class="mt-4 hidden"></div>
                </div>
            </div>

            <!-- Cycle Data Section -->
            <div class="bg-white rounded-2xl shadow-lg overflow-hidden">
                <div class="bg-pink-500 text-white px-6 py-4">
                    <h2 class="text-xl font-bold">داده‌های سایکل</h2>
                </div>
                <div class="p-6">
                    <div class="mb-4">
                        <label class="block text-sm font-medium text-gray-700 mb-2">انتخاب ماه‌ها</label>
                        <div id="monthSelector" class="flex flex-wrap gap-2"></div>
                    </div>

                    <button id="loadCycleDataBtn"
                        class="w-full bg-pink-500 hover:bg-pink-600 text-white font-bold py-3 px-6 rounded-lg transition duration-200 mb-4">
                        دریافت داده‌ها
                    </button>

                    <div id="loadingIndicator" class="hidden text-center py-8">
                        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-pink-500 mx-auto"></div>
                        <p class="mt-4 text-gray-600">در حال بارگذاری...</p>
                    </div>

                    <!-- Day Navigator -->
                    <div id="dayNavigator" class="hidden">
                        <!-- Day Header -->
                        <div class="bg-gray-800 rounded-t-xl p-4">
                            <div class="flex items-center justify-between">
                                <button id="prevDayBtn" class="bg-gray-700 hover:bg-gray-600 text-white px-4 py-2 rounded-lg transition flex items-center gap-2">
                                    <span>→</span>
                                    <span>روز قبل</span>
                                </button>
                                <div class="text-center">
                                    <div id="currentDateDisplay" class="text-white font-bold text-lg"></div>
                                    <div id="dayCountDisplay" class="text-gray-400 text-sm"></div>
                                </div>
                                <button id="nextDayBtn" class="bg-gray-700 hover:bg-gray-600 text-white px-4 py-2 rounded-lg transition flex items-center gap-2">
                                    <span>روز بعد</span>
                                    <span>←</span>
                                </button>
                            </div>
                        </div>

                        <!-- Phase Info Card -->
                        <div id="phaseInfoCard" class="bg-gradient-to-r from-gray-700 to-gray-800 p-4 border-t border-gray-600">
                            <div class="flex items-center justify-between flex-wrap gap-3">
                                <div id="phaseBadge" class="phase-badge"></div>
                                <div id="cycleDay" class="text-white text-sm"></div>
                                <div id="fertilityInfo" class="text-sm"></div>
                            </div>
                        </div>

                        <!-- JSON Display -->
                        <div class="bg-gray-900 rounded-b-xl p-4 max-h-[400px] overflow-auto">
                            <div class="flex justify-end mb-3">
                                <button id="copyDayJsonBtn" class="bg-blue-600 hover:bg-blue-700 text-white text-sm px-3 py-1 rounded transition">
                                    کپی JSON این روز
                                </button>
                            </div>
                            <div id="dayJsonViewer" class="json-container text-gray-100"></div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <script>
        const csrfToken = document.querySelector('meta[name="csrf-token"]').getAttribute('content');

        let currentCycleData = null;
        let allDays = [];
        let currentDayIndex = 0;

        // Generate month selector
        function generateMonthSelector() {
            const container = document.getElementById('monthSelector');
            const now = new Date();
            const months = [];

            for (let i = 2; i >= 1; i--) {
                months.push(new Date(now.getFullYear(), now.getMonth() - i, 1));
            }
            months.push(new Date(now.getFullYear(), now.getMonth(), 1));
            for (let i = 1; i <= 3; i++) {
                months.push(new Date(now.getFullYear(), now.getMonth() + i, 1));
            }

            const persianMonths = ['ژانویه', 'فوریه', 'مارس', 'آوریل', 'مه', 'ژوئن', 'ژوئیه', 'آگوست', 'سپتامبر', 'اکتبر', 'نوامبر', 'دسامبر'];

            container.innerHTML = months.map((d, index) => {
                const value = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
                const isCurrentMonth = d.getMonth() === now.getMonth() && d.getFullYear() === now.getFullYear();
                const checked = index >= 2 && index <= 5;
                return `
                    <label class="inline-flex items-center px-3 py-2 rounded-lg cursor-pointer transition
                        ${isCurrentMonth ? 'bg-pink-100 border-2 border-pink-500' : 'bg-gray-100 border border-gray-300'}
                        hover:bg-pink-50">
                        <input type="checkbox" value="${value}" class="month-checkbox hidden" ${checked ? 'checked' : ''}>
                        <span class="text-sm ${isCurrentMonth ? 'font-bold text-pink-700' : 'text-gray-700'}">
                            ${persianMonths[d.getMonth()]} ${d.getFullYear()}
                        </span>
                    </label>
                `;
            }).join('');

            container.querySelectorAll('.month-checkbox').forEach(cb => {
                updateCheckboxStyle(cb);
                cb.addEventListener('change', () => updateCheckboxStyle(cb));
            });
        }

        function updateCheckboxStyle(checkbox) {
            const label = checkbox.parentElement;
            if (checkbox.checked) {
                label.classList.add('ring-2', 'ring-pink-500', 'bg-pink-50');
            } else {
                label.classList.remove('ring-2', 'ring-pink-500', 'bg-pink-50');
            }
        }

        // Beautiful JSON formatter
        function formatJsonPretty(obj, indent = 0) {
            const pad = '    '.repeat(indent);
            const padInner = '    '.repeat(indent + 1);

            if (obj === null) {
                return `<span class="json-null">null</span>`;
            }
            if (typeof obj === 'boolean') {
                return `<span class="json-boolean">${obj}</span>`;
            }
            if (typeof obj === 'number') {
                return `<span class="json-number">${obj}</span>`;
            }
            if (typeof obj === 'string') {
                const escaped = obj.replace(/"/g, '\\"').replace(/\n/g, '\\n');
                return `<span class="json-string">"${escapeHtml(escaped)}"</span>`;
            }

            if (Array.isArray(obj)) {
                if (obj.length === 0) return '<span class="json-bracket">[]</span>';
                const items = obj.map(item => `<div class="json-row">${padInner}${formatJsonPretty(item, indent + 1)}</div>`);
                return `<span class="json-bracket">[</span>\n${items.join(',\n')}\n${pad}<span class="json-bracket">]</span>`;
            }

            if (typeof obj === 'object') {
                const keys = Object.keys(obj);
                if (keys.length === 0) return '<span class="json-bracket">{}</span>';

                const items = keys.map(key => {
                    const val = formatJsonPretty(obj[key], indent + 1);
                    return `<div class="json-row">${padInner}<span class="json-key">"${escapeHtml(key)}"</span>: ${val}</div>`;
                });

                return `<span class="json-bracket">{</span>\n${items.join(',\n')}\n${pad}<span class="json-bracket">}</span>`;
            }

            return String(obj);
        }

        function escapeHtml(str) {
            return String(str).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
        }

        // Get phase color
        function getPhaseColor(phase) {
            const colors = {
                'menstruation': { bg: '#dc2626', text: 'white', label: 'قاعدگی' },
                'follicular': { bg: '#16a34a', text: 'white', label: 'فولیکولار' },
                'ovulation': { bg: '#9333ea', text: 'white', label: 'تخمک‌گذاری' },
                'luteal': { bg: '#ea580c', text: 'white', label: 'لوتئال' },
            };
            return colors[phase] || { bg: '#6b7280', text: 'white', label: phase };
        }

        // Display current day
        function displayCurrentDay() {
            if (allDays.length === 0) return;

            const dayData = allDays[currentDayIndex];
            const dateStr = dayData.date;
            const data = dayData.data;

            // Update header
            document.getElementById('currentDateDisplay').textContent = dateStr;
            document.getElementById('dayCountDisplay').textContent = `روز ${currentDayIndex + 1} از ${allDays.length}`;

            // Update phase badge
            const phaseInfo = getPhaseColor(data.phase);
            const phaseBadge = document.getElementById('phaseBadge');
            phaseBadge.style.backgroundColor = phaseInfo.bg;
            phaseBadge.style.color = phaseInfo.text;
            phaseBadge.textContent = data.phase_label || phaseInfo.label;

            // Update cycle day
            document.getElementById('cycleDay').textContent = `روز ${data.cycle_day || '-'} سایکل`;

            // Update fertility info
            const fertilityDiv = document.getElementById('fertilityInfo');
            if (data.is_fertile_window) {
                fertilityDiv.innerHTML = `<span class="bg-green-500 text-white px-3 py-1 rounded-full text-xs">پنجره باروری</span>`;
            } else if (data.is_pms_window) {
                fertilityDiv.innerHTML = `<span class="bg-orange-500 text-white px-3 py-1 rounded-full text-xs">PMS</span>`;
            } else {
                fertilityDiv.innerHTML = '';
            }

            // Update JSON viewer
            document.getElementById('dayJsonViewer').innerHTML = formatJsonPretty(data);

            // Update button states
            document.getElementById('prevDayBtn').disabled = currentDayIndex === 0;
            document.getElementById('nextDayBtn').disabled = currentDayIndex === allDays.length - 1;

            document.getElementById('prevDayBtn').classList.toggle('opacity-50', currentDayIndex === 0);
            document.getElementById('nextDayBtn').classList.toggle('opacity-50', currentDayIndex === allDays.length - 1);
        }

        // Save profile
        document.getElementById('profileForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const btn = document.getElementById('saveProfileBtn');
            btn.disabled = true;
            btn.textContent = 'در حال ذخیره...';

            const formData = new FormData(e.target);
            const data = Object.fromEntries(formData.entries());

            try {
                const response = await fetch('/test-page/profile', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-CSRF-TOKEN': csrfToken,
                        'Accept': 'application/json',
                    },
                    body: JSON.stringify(data),
                });

                const result = await response.json();
                const statusDiv = document.getElementById('profileStatus');

                if (result.success) {
                    statusDiv.innerHTML = '<div class="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded-lg">پروفایل با موفقیت ذخیره شد</div>';
                } else {
                    statusDiv.innerHTML = '<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-lg">خطا در ذخیره پروفایل</div>';
                }
                statusDiv.classList.remove('hidden');
                setTimeout(() => statusDiv.classList.add('hidden'), 3000);
            } catch (error) {
                console.error('Error:', error);
            } finally {
                btn.disabled = false;
                btn.textContent = 'ذخیره پروفایل';
            }
        });

        // Load cycle data
        document.getElementById('loadCycleDataBtn').addEventListener('click', async () => {
            const selectedMonths = Array.from(document.querySelectorAll('.month-checkbox:checked')).map(cb => cb.value);

            if (selectedMonths.length === 0) {
                alert('لطفاً حداقل یک ماه انتخاب کنید');
                return;
            }

            const loadingDiv = document.getElementById('loadingIndicator');
            const dayNavigator = document.getElementById('dayNavigator');

            loadingDiv.classList.remove('hidden');
            dayNavigator.classList.add('hidden');

            try {
                const response = await fetch('/test-page/cycle-data', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-CSRF-TOKEN': csrfToken,
                        'Accept': 'application/json',
                    },
                    body: JSON.stringify({ months: selectedMonths, locale: 'fa' }),
                });

                const result = await response.json();
                currentCycleData = result.data;

                // Flatten all days
                allDays = [];
                Object.keys(result.data).sort().forEach(monthKey => {
                    const monthData = result.data[monthKey];
                    Object.keys(monthData.days).sort().forEach(dateKey => {
                        allDays.push({
                            date: dateKey,
                            data: monthData.days[dateKey]
                        });
                    });
                });

                // Find today or start from beginning
                const today = new Date().toISOString().split('T')[0];
                currentDayIndex = allDays.findIndex(d => d.date === today);
                if (currentDayIndex === -1) currentDayIndex = 0;

                dayNavigator.classList.remove('hidden');
                displayCurrentDay();
            } catch (error) {
                console.error('Error:', error);
                alert('خطا در دریافت داده‌ها');
            } finally {
                loadingDiv.classList.add('hidden');
            }
        });

        // Navigation buttons
        document.getElementById('prevDayBtn').addEventListener('click', () => {
            if (currentDayIndex > 0) {
                currentDayIndex--;
                displayCurrentDay();
            }
        });

        document.getElementById('nextDayBtn').addEventListener('click', () => {
            if (currentDayIndex < allDays.length - 1) {
                currentDayIndex++;
                displayCurrentDay();
            }
        });

        // Copy current day JSON
        document.getElementById('copyDayJsonBtn').addEventListener('click', () => {
            if (allDays.length > 0) {
                const dayData = allDays[currentDayIndex];
                navigator.clipboard.writeText(JSON.stringify(dayData.data, null, 2));
                alert('JSON کپی شد!');
            }
        });

        // Keyboard navigation
        document.addEventListener('keydown', (e) => {
            if (allDays.length === 0) return;
            if (e.key === 'ArrowLeft' && currentDayIndex < allDays.length - 1) {
                currentDayIndex++;
                displayCurrentDay();
            } else if (e.key === 'ArrowRight' && currentDayIndex > 0) {
                currentDayIndex--;
                displayCurrentDay();
            }
        });

        // Initialize
        generateMonthSelector();
    </script>
</body>
</html>
