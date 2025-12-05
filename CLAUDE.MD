# CLAUDE.MD

## قوانین کدنویسی (الزامی)

**SOLID + DRY + KISS + YAGNI** در همه جا رعایت شود.

## API

- ورژن‌بندی: `/api/v1/*`
- هر endpoint جدید **باید** در Swagger مستند شود (`@OA\`)

## معماری: DDD

```
app/
├── Domain/           # Entities, ValueObjects, Repository Interfaces
├── Application/      # DTOs, UseCases, Services
└── Infrastructure/   # Repository Implementations
```
