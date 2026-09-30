# Canvas v1 — design snapshot

Snapshot of the Claude Design canvas **«ریتمی — یائسگی»** (https://claude.ai/artifact/RT9a6Xy7hG9azjjwZnQnBs,
version 1790787541-747c, taken 2026-09-30). Source of truth for the `/canvas-build` queue (`roadmap/`).

- `boards/*.dc.html` — artboard sources. `nbl_*` = light, `nbd_*` = dark (same content; only Nav/Shop exist as dark only),
  `W_*` = desktop web (1440 px), `IA_*` = information-architecture boards, `Main.dc.html` = menopause home (dark).
  They are static HTML with inline styles: open with `file://` in a browser/Playwright at the board width (390 or 1440)
  to see them (the `support.js` 404 is harmless).
- `text/*.txt` — readable text + link targets of every board (`python3 extract.py boards/X.dc.html`).
- **Colors in the boards are NOT the spec** — build with the app's current light + dark tokens (frontend/CLAUDE.md §10.2).
  Layout, hierarchy, copy, components and flows ARE the spec.

| Page | Epic | Board (file · title · size) |
|---|---|---|
| ساختار و ناوبری | NAV | `IA_Nav.dc.html` · قواعد ناوبری پایین · 1440×1550 |
|  |  | `IA_Map.dc.html` · نقشه ساختار اپ · 1440×1280 |
|  |  | `nbd_Nav_Today.dc.html` · ۱ · امروز (هدر با چیپ حالت و جست‌وجو) · 390×1260 |
|  |  | `nbd_Nav_Stage.dc.html` · ۲ · تب مرحله: تقویم و تحلیل · 390×980 |
|  |  | `nbd_Nav_Plus.dc.html` · ۳ · برگه ثبت (+) · 390×844 |
|  |  | `nbd_Nav_Services.dc.html` · ۴ · تب خدمات · 390×1670 |
|  |  | `nbd_Nav_Me.dc.html` · ۵ · تب من · 390×1940 |
|  |  | `nbd_Nav_Mode.dc.html` · ۶ · انتخاب مرحله زندگی · 390×1180 |
|  |  | `nbd_Nav_Search.dc.html` · ۷ · جست‌وجوی سراسری · 390×1050 |
| یائسگی | MENO | `Main.dc.html` · ۱ · خانه یائسگی · 390×1890 |
|  |  | `nbl_Meno_Home.dc.html` · ۱ · خانه یائسگی · 390×1890 |
|  |  | `nbl_Meno_Stage.dc.html` · ۲ · انتخاب مرحله · 390×1080 |
|  |  | `nbl_Meno_Log.dc.html` · ۳ · ثبت علائم روزانه · 390×2260 |
|  |  | `nbl_Meno_HotFlash.dc.html` · ۴ · ثبت گرگرفتگی · 390×1320 |
|  |  | `nbl_Meno_Alert.dc.html` · ۵ · هشدار خونریزی و علائم خطر · 390×1420 |
|  |  | `nbl_Meno_Score.dc.html` · ۶ · امتیاز علائم (ماهانه) · 390×1590 |
|  |  | `nbl_Meno_Checkups.dc.html` · ۷ · چکاپ‌ها و آزمایش‌ها · 390×1760 |
|  |  | `nbl_Meno_Treatment.dc.html` · ۸ · درمان و مراقبت · 390×1720 |
|  |  | `nbl_Meno_Report.dc.html` · ۹ · گزارش برای پزشک · 390×1880 |
| درمان ناباروری (IVF) | IVF | `nbl_IVF_Home.dc.html` · ۱ · خانه درمان · 390×1670 |
|  |  | `nbl_IVF_Meds.dc.html` · ۲ · برنامه تزریق · 390×1610 |
|  |  | `nbl_IVF_Scan.dc.html` · ۳ · ثبت سونو · 390×1400 |
|  |  | `nbl_IVF_TWW.dc.html` · ۴ · دو هفته انتظار · 390×1320 |
| سقط و از دست دادن بارداری | LOSS | `nbl_Loss_Start.dc.html` · ۱ · ثبت با لحن آرام · 390×1260 |
|  |  | `nbl_Loss_Care.dc.html` · ۲ · مراقبت و حمایت · 390×1500 |
|  |  | `nbl_Loss_Next.dc.html` · ۳ · قدم بعدی · 390×1000 |
| برنامه‌های بیماری‌ها | COND | `nbl_Cond_Hub.dc.html` · ۱ · برنامه‌ها · 390×1290 |
|  |  | `nbl_Cond_Endo.dc.html` · ۲ · دفترچه درد اندومتریوز · 390×1640 |
|  |  | `nbl_Cond_PMDD.dc.html` · ۳ · پرسشنامه PMDD · 390×1560 |
|  |  | `nbl_Cond_Bleed.dc.html` · ۴ · جدول خونریزی · 390×1460 |
| پیشگیری از بارداری | CONTRA | `nbl_Contra_Setup.dc.html` · ۱ · انتخاب روش · 390×1200 |
|  |  | `nbl_Contra_Pill.dc.html` · ۲ · بسته قرص · 390×1380 |
|  |  | `nbl_Contra_Missed.dc.html` · ۳ · قرص جا افتاده · 390×1300 |
|  |  | `nbl_Contra_Other.dc.html` · ۴ · آی‌یو‌دی، آمپول، کاشتنی · 390×1260 |
| اتصال به ساعت و گجت‌ها | (dropped) | `nbl_Wear_Connect.dc.html` · ۱ · منبع‌ها · 390×1610 |
|  |  | `nbl_Wear_Data.dc.html` · ۲ · داده‌های دریافتی · 390×1260 |
| قفل و حریم خصوصی | PRIV | `nbl_Priv_Settings.dc.html` · ۱ · تنظیمات · 390×1430 |
|  |  | `nbl_Priv_Lock.dc.html` · ۲ · صفحه قفل · 390×844 |
|  |  | `nbl_Priv_Icon.dc.html` · ۳ · آیکون خنثی · 390×1020 |
| حالت نوجوان | TEEN | `nbl_Teen_Onb.dc.html` · ۱ · شروع · 390×1100 |
|  |  | `nbl_Teen_Home.dc.html` · ۲ · خانه نوجوان · 390×1670 |
|  |  | `nbl_Teen_Parent.dc.html` · ۳ · همراهی مادر · 390×1060 |
| کف لگن و مثانه | PELV | `nbl_Pelvic_Plan.dc.html` · ۱ · برنامه و دفترچه · 390×1410 |
|  |  | `nbl_Pelvic_Kegel.dc.html` · ۲ · تمرین کگل · 390×1000 |
| بیمه | INS | `nbl_Ins_Home.dc.html` · ۱ · خانه بیمه · 390×1710 |
|  |  | `nbl_Ins_Coverage.dc.html` · ۲ · پوشش‌ها و سقف‌ها · 390×1500 |
|  |  | `nbl_Ins_Health.dc.html` · ۳ · پرسشنامه سلامت · 390×1640 |
|  |  | `nbl_Ins_Status.dc.html` · ۴ · وضعیت درخواست‌ها · 390×1240 |
|  |  | `nbl_Ins_Claims.dc.html` · ۵ · سوابق خسارت · 390×1600 |
|  |  | `nbl_Ins_ClaimNew.dc.html` · ۶ · ثبت خسارت · 390×1560 |
|  |  | `nbl_Ins_ClaimDetail.dc.html` · ۷ · جزئیات خسارت · 390×1480 |
|  |  | `nbl_Ins_Centers.dc.html` · ۸ · مراکز طرف قرارداد · 390×1560 |
| پرونده سلامت | REC | `nbl_Rec_Home.dc.html` · ۱ · خلاصه پرونده · 390×1650 |
|  |  | `nbl_Rec_Timeline.dc.html` · ۲ · سوابق و اسناد · 390×1360 |
|  |  | `nbl_Rec_Doc.dc.html` · ۳ · جزئیات سند · 390×1320 |
|  |  | `nbl_Rec_Share.dc.html` · ۴ · دسترسی‌ها · 390×1360 |
|  |  | `nbl_Rec_Emergency.dc.html` · ۵ · کارت اضطراری · 390×1240 |
| ثبت با صدا | VOICE | `nbl_Voice_Entry.dc.html` · ۱ · انتخاب ثبت با صدا · 390×1360 |
|  |  | `nbl_Voice_Record.dc.html` · ۲ · ضبط و متن زنده · 390×844 |
|  |  | `nbl_Voice_Review.dc.html` · ۳ · بررسی و ویرایش · 390×1900 |
|  |  | `nbl_Voice_Saved.dc.html` · ۴ · ثبت شد · 390×1300 |
| خدمات مادر و کودک | DIR | `nbl_Dir_Home.dc.html` · ۱ · خانه خدمات · 390×1730 |
|  |  | `nbl_Dir_JoinDocs.dc.html` · ۱۰ · ثبت مجموعه: خدمات و مدارک · 390×1350 |
|  |  | `nbl_Dir_JoinDone.dc.html` · ۱۱ · درخواست در حال بررسی · 390×950 |
|  |  | `nbl_Dir_Map.dc.html` · ۲ · نقشه · 390×844 |
|  |  | `nbl_Dir_List.dc.html` · ۳ · فهرست و فیلتر · 390×1210 |
|  |  | `nbl_Dir_Place.dc.html` · ۴ · صفحه مجموعه · 390×2580 |
|  |  | `nbl_Dir_Book.dc.html` · ۵ · انتخاب وقت · 390×1300 |
|  |  | `nbl_Dir_Booked.dc.html` · ۶ · رزرو ثبت شد · 390×1210 |
|  |  | `nbl_Dir_MyBookings.dc.html` · ۷ · رزروهای من · 390×1000 |
|  |  | `nbl_Dir_Join.dc.html` · ۸ · ثبت مجموعه: معرفی · 390×1350 |
|  |  | `nbl_Dir_JoinForm.dc.html` · ۹ · ثبت مجموعه: مکان و تصاویر · 390×1830 |
|  |  | `W_Dir_Search.dc.html` · وب ۱ · جست‌وجو و نقشه · 1440×2348 |
|  |  | `W_Dir_Place.dc.html` · وب ۲ · صفحه مجموعه و رزرو · 1440×3246 |
|  |  | `W_Dir_Booked.dc.html` · وب ۳ · رزرو ثبت شد · 1440×1118 |
|  |  | `W_Dir_Business.dc.html` · وب ۴ · صفحه کسب‌وکارها · 1440×2978 |
|  |  | `W_Dir_Join.dc.html` · وب ۵ · فرم ثبت مجموعه · 1440×2358 |
|  |  | `W_Dir_JoinDone.dc.html` · وب ۶ · درخواست در حال بررسی · 1440×1049 |
| فروشگاه | SHOP | `nbd_Shop_Home.dc.html` · ۱ · فروشگاه سیسمونی و نوزاد · 390×2070 |
|  |  | `nbd_Shop_Beauty.dc.html` · ۲ · فروشگاه آرایشی و بهداشتی · 390×1770 |
|  |  | `nbd_Shop_List.dc.html` · ۳ · فهرست و فیلتر · 390×1270 |
|  |  | `nbd_Shop_Product.dc.html` · ۴ · صفحه محصول نوزاد · 390×1780 |
|  |  | `nbd_Shop_ProductBeauty.dc.html` · ۵ · صفحه محصول آرایشی · 390×1420 |
|  |  | `nbd_Shop_Checklist.dc.html` · ۶ · لیست سیسمونی · 390×1290 |
|  |  | `nbd_Shop_Cart.dc.html` · ۷ · سبد خرید · 390×990 |
|  |  | `nbd_Shop_Checkout.dc.html` · ۸ · ارسال و پرداخت · 390×1320 |
|  |  | `nbd_Shop_Order.dc.html` · ۹ · سفارش ثبت شد · 390×1150 |
|  |  | `W_Shop_Home.dc.html` · وب ۱ · خانه فروشگاه · 1440×2750 |
|  |  | `W_Shop_List.dc.html` · وب ۲ · فهرست و فیلتر · 1440×1708 |
|  |  | `W_Shop_Product.dc.html` · وب ۳ · صفحه محصول · 1440×2261 |
|  |  | `W_Shop_Cart.dc.html` · وب ۴ · سبد خرید · 1440×1560 |
|  |  | `W_Shop_Checkout.dc.html` · وب ۵ · ارسال و پرداخت · 1440×1324 |
|  |  | `W_Shop_Done.dc.html` · وب ۶ · سفارش ثبت شد · 1440×1268 |
