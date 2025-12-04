# CLAUDE.MD - راهنمای توسعه پروژه

## API Versioning (ورژن‌بندی API)

این پروژه از ورژن‌بندی API استفاده می‌کند. تمام endpointها باید تحت پیشوند ورژن قرار بگیرند:

- `/api/v1/*` - نسخه اول API

هنگام افزودن endpoint جدید، حتماً آن را در ورژن مناسب قرار دهید.

---

## Swagger Documentation

**مهم:** هر API جدیدی که اضافه می‌شود، **حتماً** باید مستندات Swagger داشته باشد.

- از annotation های `@OA\` برای مستندسازی استفاده کنید
- تمام پارامترهای ورودی، خروجی و کدهای وضعیت را مستند کنید
- مثال برای request و response اضافه کنید

---

## DDD (Domain-Driven Design)

این پروژه از الگوی طراحی **DDD** پیروی می‌کند. ساختار پروژه باید به صورت زیر باشد:

```
app/
├── Domain/           # منطق کسب و کار
│   ├── Entities/     # موجودیت‌ها
│   ├── ValueObjects/ # آبجکت‌های ارزشی
│   ├── Repositories/ # اینترفیس ریپازیتوری‌ها
│   └── Services/     # سرویس‌های دامنه
├── Application/      # لایه اپلیکیشن
│   ├── DTOs/         # آبجکت‌های انتقال داده
│   ├── UseCases/     # موارد استفاده
│   └── Services/     # سرویس‌های اپلیکیشن
└── Infrastructure/   # لایه زیرساخت
    ├── Repositories/ # پیاده‌سازی ریپازیتوری‌ها
    └── Persistence/  # پایداری داده
```

---

## SOLID Principles (اصول SOLID) ⚠️ بسیار مهم

رعایت اصول SOLID **الزامی** است:

### S - Single Responsibility Principle
هر کلاس فقط یک مسئولیت داشته باشد.

### O - Open/Closed Principle
کلاس‌ها برای توسعه باز و برای تغییر بسته باشند.

### L - Liskov Substitution Principle
کلاس‌های فرزند باید جایگزین کلاس‌های والد شوند بدون تغییر رفتار.

### I - Interface Segregation Principle
اینترفیس‌های کوچک و تخصصی به جای اینترفیس‌های بزرگ.

### D - Dependency Inversion Principle
وابستگی به abstraction نه implementation.

---

## چک‌لیست قبل از Commit

- [ ] آیا API جدید در Swagger مستند شده است؟
- [ ] آیا ورژن‌بندی API رعایت شده است؟
- [ ] آیا ساختار DDD رعایت شده است؟
- [ ] آیا اصول SOLID رعایت شده است؟
