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
        .json-viewer {
            font-family: 'Courier New', monospace;
            font-size: 12px;
            direction: ltr;
            text-align: left;
        }
        .json-key { color: #0550ae; }
        .json-string { color: #0a3069; }
        .json-number { color: #0550ae; }
        .json-boolean { color: #cf222e; }
        .json-null { color: #6e7781; }
    </style>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto px-4 py-8 max-w-6xl">
        <!-- Header -->
        <div class="bg-gradient-to-r from-pink-500 to-purple-600 rounded-2xl p-6 mb-8 text-white shadow-lg">
            <h1 class="text-3xl font-bold mb-2">🧪 صفحه تست</h1>
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
                    <h2 class="text-xl font-bold">👤 پروفایل کاربر</h2>
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
                            💾 ذخیره پروفایل
                        </button>
                    </form>

                    <div id="profileStatus" class="mt-4 hidden">
                        <div class="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded-lg">
                            پروفایل با موفقیت ذخیره شد
                        </div>
                    </div>
                </div>
            </div>

            <!-- Cycle Data Section -->
            <div class="bg-white rounded-2xl shadow-lg overflow-hidden">
                <div class="bg-pink-500 text-white px-6 py-4">
                    <h2 class="text-xl font-bold">📊 داده‌های سایکل</h2>
                </div>
                <div class="p-6">
                    <div class="mb-4">
                        <label class="block text-sm font-medium text-gray-700 mb-2">انتخاب ماه‌ها</label>
                        <div id="monthSelector" class="flex flex-wrap gap-2">
                            <!-- Months will be generated by JS -->
                        </div>
                    </div>

                    <div class="flex gap-2 mb-4">
                        <button id="loadCycleDataBtn"
                            class="flex-1 bg-pink-500 hover:bg-pink-600 text-white font-bold py-3 px-6 rounded-lg transition duration-200">
                            📥 دریافت داده‌ها
                        </button>
                        <button id="copyJsonBtn"
                            class="bg-gray-500 hover:bg-gray-600 text-white font-bold py-3 px-4 rounded-lg transition duration-200">
                            📋
                        </button>
                    </div>

                    <div id="loadingIndicator" class="hidden text-center py-8">
                        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-pink-500 mx-auto"></div>
                        <p class="mt-4 text-gray-600">در حال بارگذاری...</p>
                    </div>

                    <div id="cycleDataContainer" class="hidden">
                        <div class="bg-gray-50 rounded-lg p-4 max-h-96 overflow-auto">
                            <pre id="cycleJsonViewer" class="json-viewer whitespace-pre-wrap"></pre>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Full JSON View -->
        <div id="fullJsonSection" class="mt-8 bg-white rounded-2xl shadow-lg overflow-hidden hidden">
            <div class="bg-gray-800 text-white px-6 py-4 flex justify-between items-center">
                <h2 class="text-xl font-bold">📋 نمایش کامل JSON</h2>
                <div class="flex gap-2">
                    <button id="toggleFullJsonBtn" class="bg-gray-600 hover:bg-gray-700 px-4 py-2 rounded-lg text-sm">
                        جمع/باز کردن
                    </button>
                    <button id="copyFullJsonBtn" class="bg-blue-500 hover:bg-blue-600 px-4 py-2 rounded-lg text-sm">
                        کپی کل JSON
                    </button>
                </div>
            </div>
            <div id="fullJsonContainer" class="p-6">
                <div class="bg-gray-900 rounded-lg p-4 max-h-[600px] overflow-auto">
                    <pre id="fullJsonViewer" class="json-viewer text-gray-100"></pre>
                </div>
            </div>
        </div>
    </div>

    <script>
        // CSRF Token for requests
        const csrfToken = document.querySelector('meta[name="csrf-token"]').getAttribute('content');

        // Current cycle data
        let currentCycleData = null;

        // Generate month selector buttons
        function generateMonthSelector() {
            const container = document.getElementById('monthSelector');
            const now = new Date();

            const months = [];
            // 2 ماه قبل
            for (let i = 2; i >= 1; i--) {
                const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
                months.push(d);
            }
            // ماه جاری
            months.push(new Date(now.getFullYear(), now.getMonth(), 1));
            // 3 ماه بعد
            for (let i = 1; i <= 3; i++) {
                const d = new Date(now.getFullYear(), now.getMonth() + i, 1);
                months.push(d);
            }

            const persianMonths = ['ژانویه', 'فوریه', 'مارس', 'آوریل', 'مه', 'ژوئن', 'ژوئیه', 'آگوست', 'سپتامبر', 'اکتبر', 'نوامبر', 'دسامبر'];

            container.innerHTML = months.map((d, index) => {
                const value = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
                const isCurrentMonth = d.getMonth() === now.getMonth() && d.getFullYear() === now.getFullYear();
                const checked = index >= 2 && index <= 5; // ماه جاری و 3 ماه بعد انتخاب شده باشند
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

            // Update checkbox styling
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

        // Format JSON with syntax highlighting
        function formatJson(obj, indent = 0) {
            const spaces = '  '.repeat(indent);

            if (obj === null) {
                return '<span class="json-null">null</span>';
            }

            if (typeof obj === 'boolean') {
                return `<span class="json-boolean">${obj}</span>`;
            }

            if (typeof obj === 'number') {
                return `<span class="json-number">${obj}</span>`;
            }

            if (typeof obj === 'string') {
                return `<span class="json-string">"${escapeHtml(obj)}"</span>`;
            }

            if (Array.isArray(obj)) {
                if (obj.length === 0) return '[]';
                const items = obj.map(item => spaces + '  ' + formatJson(item, indent + 1));
                return '[\n' + items.join(',\n') + '\n' + spaces + ']';
            }

            if (typeof obj === 'object') {
                const keys = Object.keys(obj);
                if (keys.length === 0) return '{}';
                const items = keys.map(key =>
                    spaces + '  ' + `<span class="json-key">"${escapeHtml(key)}"</span>: ` + formatJson(obj[key], indent + 1)
                );
                return '{\n' + items.join(',\n') + '\n' + spaces + '}';
            }

            return String(obj);
        }

        function escapeHtml(str) {
            return str.replace(/&/g, '&amp;')
                      .replace(/</g, '&lt;')
                      .replace(/>/g, '&gt;')
                      .replace(/"/g, '&quot;');
        }

        // Save profile
        document.getElementById('profileForm').addEventListener('submit', async (e) => {
            e.preventDefault();

            const btn = document.getElementById('saveProfileBtn');
            btn.disabled = true;
            btn.textContent = '⏳ در حال ذخیره...';

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
                    statusDiv.innerHTML = '<div class="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded-lg">✅ پروفایل با موفقیت ذخیره شد</div>';
                } else {
                    statusDiv.innerHTML = '<div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-lg">❌ خطا در ذخیره پروفایل</div>';
                }
                statusDiv.classList.remove('hidden');
                setTimeout(() => statusDiv.classList.add('hidden'), 3000);
            } catch (error) {
                console.error('Error:', error);
            } finally {
                btn.disabled = false;
                btn.textContent = '💾 ذخیره پروفایل';
            }
        });

        // Load cycle data
        document.getElementById('loadCycleDataBtn').addEventListener('click', async () => {
            const selectedMonths = Array.from(document.querySelectorAll('.month-checkbox:checked'))
                .map(cb => cb.value);

            if (selectedMonths.length === 0) {
                alert('لطفاً حداقل یک ماه انتخاب کنید');
                return;
            }

            const loadingDiv = document.getElementById('loadingIndicator');
            const dataContainer = document.getElementById('cycleDataContainer');
            const fullJsonSection = document.getElementById('fullJsonSection');

            loadingDiv.classList.remove('hidden');
            dataContainer.classList.add('hidden');
            fullJsonSection.classList.add('hidden');

            try {
                const response = await fetch('/test-page/cycle-data', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-CSRF-TOKEN': csrfToken,
                        'Accept': 'application/json',
                    },
                    body: JSON.stringify({
                        months: selectedMonths,
                        locale: 'fa',
                    }),
                });

                const result = await response.json();
                currentCycleData = result.data;

                // Show summary in small viewer
                const summary = {
                    total_months: Object.keys(result.data).length,
                    months: Object.keys(result.data),
                    sample_day: result.data[Object.keys(result.data)[0]]?.days[Object.keys(result.data[Object.keys(result.data)[0]]?.days || {})[0]] || null,
                };
                document.getElementById('cycleJsonViewer').innerHTML = formatJson(summary);

                // Show full data
                document.getElementById('fullJsonViewer').innerHTML = formatJson(result.data);

                dataContainer.classList.remove('hidden');
                fullJsonSection.classList.remove('hidden');
            } catch (error) {
                console.error('Error:', error);
                alert('خطا در دریافت داده‌ها');
            } finally {
                loadingDiv.classList.add('hidden');
            }
        });

        // Copy JSON buttons
        document.getElementById('copyJsonBtn').addEventListener('click', () => {
            if (currentCycleData) {
                navigator.clipboard.writeText(JSON.stringify(currentCycleData, null, 2));
                alert('JSON کپی شد!');
            }
        });

        document.getElementById('copyFullJsonBtn').addEventListener('click', () => {
            if (currentCycleData) {
                navigator.clipboard.writeText(JSON.stringify(currentCycleData, null, 2));
                alert('JSON کپی شد!');
            }
        });

        // Toggle full JSON view
        let isFullJsonCollapsed = false;
        document.getElementById('toggleFullJsonBtn').addEventListener('click', () => {
            const container = document.getElementById('fullJsonContainer');
            isFullJsonCollapsed = !isFullJsonCollapsed;
            container.style.display = isFullJsonCollapsed ? 'none' : 'block';
        });

        // Initialize
        generateMonthSelector();
    </script>
</body>
</html>
