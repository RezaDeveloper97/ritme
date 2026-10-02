// Trimmed copies of backend-go/contract/golden menopause/today_flow, today_empty, profile_flow and
// messages/menopause_flow (CB-MENO-02 / CB-MENO-12) — re-copy when those goldens change.

export const TODAY_FLOW: unknown = {
  "date": "2026-09-23",
  "profile": {
    "stage": "meno",
    "stored_stage": "meno",
    "last_period": "2025-08-01",
    "surgical": null,
    "hrt": null,
    "months_without_period": 13,
    "suggested_stage": null,
    "needs_stage": false,
    "post_menopausal": true,
    "tip": {
      "code": "stage_meno",
      "title": "Menopause",
      "body": "It has been more than 12 months, which means you have reached menopause. From now on, log any bleeding or spotting and tell your doctor.",
      "meta": {
        "placement": "home",
        "stages": [
          "meno"
        ]
      },
      "audiences": [
        "menopause"
      ],
      "needs_review": true
    }
  },
  "hot_flashes": {
    "count": 2,
    "night_count": 1,
    "avg_duration_s": 150,
    "running": null
  },
  "night_sweats": {
    "count": 1,
    "level": null
  },
  "sleep": null,
  "score": {
    "max": 44,
    "latest": {
      "month": "2026-09-23",
      "jalali_year": 1405,
      "jalali_month": 7,
      "total": 14,
      "max": 44,
      "band": {
        "code": "moderate",
        "title": "Moderate",
        "min": 9,
        "max": 16
      },
      "domains": [
        {
          "code": "somatic",
          "score": 7,
          "max": 16
        },
        {
          "code": "psychological",
          "score": 4,
          "max": 16
        },
        {
          "code": "urogenital",
          "score": 3,
          "max": 12
        }
      ],
      "delta": null,
      "previous_month": null
    },
    "trend": [
      {
        "month": "2026-04-21",
        "jalali_year": 1405,
        "jalali_month": 2,
        "total": null,
        "band": null
      },
      {
        "month": "2026-05-22",
        "jalali_year": 1405,
        "jalali_month": 3,
        "total": null,
        "band": null
      },
      {
        "month": "2026-06-22",
        "jalali_year": 1405,
        "jalali_month": 4,
        "total": null,
        "band": null
      },
      {
        "month": "2026-07-23",
        "jalali_year": 1405,
        "jalali_month": 5,
        "total": null,
        "band": null
      },
      {
        "month": "2026-08-23",
        "jalali_year": 1405,
        "jalali_month": 6,
        "total": null,
        "band": null
      },
      {
        "month": "2026-09-23",
        "jalali_year": 1405,
        "jalali_month": 7,
        "total": 14,
        "band": "moderate"
      }
    ]
  },
  "checkups": [
    {
      "id": 3,
      "key": "pap_smear",
      "title": "Pap smear / HPV",
      "subtitle": "Cervical screening",
      "category": "multi_year",
      "section": "this_month",
      "status": "due",
      "icon": "shield",
      "tone": "violet",
      "interval_label": "Every 3 years",
      "timing_label": "Cycle day 10–20",
      "last_done_on": null,
      "next_due_on": null,
      "next_due_label": "Due now",
      "is_custom": false
    }
  ],
  "treatment": [],
  "bleeding": {
    "alert": true,
    "last_on": "2026-09-15",
    "window_days": 30,
    "alert_item": {
      "code": "postmenopausal_bleeding",
      "title": "Bleeding after menopause",
      "body": "Any bleeding or spotting, even a little, 12 months or more after your last period should be checked. Usually the cause is simple, but an early check matters.",
      "meta": {
        "severity": "urgent",
        "primary": true,
        "stages": [
          "meno",
          "post"
        ],
        "cta": "Tell your doctor about this"
      },
      "audiences": [
        "menopause"
      ],
      "needs_review": true
    }
  }
};

export const TODAY_EMPTY: unknown = {
  "date": "2026-09-23",
  "profile": {
    "stage": null,
    "stored_stage": null,
    "last_period": null,
    "surgical": null,
    "hrt": null,
    "months_without_period": null,
    "suggested_stage": null,
    "needs_stage": true,
    "post_menopausal": false,
    "tip": null
  },
  "hot_flashes": {
    "count": 0,
    "night_count": 0,
    "avg_duration_s": null,
    "running": null
  },
  "night_sweats": {
    "count": 0,
    "level": null
  },
  "sleep": null,
  "score": {
    "max": 44,
    "latest": null,
    "trend": [
      {
        "month": "2026-04-21",
        "jalali_year": 1405,
        "jalali_month": 2,
        "total": null,
        "band": null
      },
      {
        "month": "2026-05-22",
        "jalali_year": 1405,
        "jalali_month": 3,
        "total": null,
        "band": null
      },
      {
        "month": "2026-06-22",
        "jalali_year": 1405,
        "jalali_month": 4,
        "total": null,
        "band": null
      },
      {
        "month": "2026-07-23",
        "jalali_year": 1405,
        "jalali_month": 5,
        "total": null,
        "band": null
      },
      {
        "month": "2026-08-23",
        "jalali_year": 1405,
        "jalali_month": 6,
        "total": null,
        "band": null
      },
      {
        "month": "2026-09-23",
        "jalali_year": 1405,
        "jalali_month": 7,
        "total": null,
        "band": null
      }
    ]
  },
  "checkups": [],
  "treatment": [],
  "bleeding": {
    "alert": false,
    "last_on": "2026-09-15",
    "window_days": 30,
    "alert_item": null
  }
};

export const PROFILE_PERI: unknown = {
  "stage": "peri",
  "stored_stage": "peri",
  "last_period": "2025-08-01",
  "surgical": false,
  "hrt": null,
  "months_without_period": 13,
  "suggested_stage": "meno",
  "needs_stage": false,
  "post_menopausal": false,
  "tip": {
    "code": "stage_peri",
    "title": "Perimenopause",
    "body": "Periods may become irregular, lighter or heavier, and new symptoms may start. Logging periods and symptoms helps you see patterns and talk to your doctor in detail.",
    "meta": {
      "placement": "home",
      "stages": [
        "peri"
      ]
    },
    "audiences": [
      "menopause"
    ],
    "needs_review": true
  }
};

export const MESSAGES_FLOW: unknown = {
  "mode": "menopause",
  "stage": "meno",
  "messages": [
    {
      "key": "postmenopausal_bleeding",
      "kind": "alert",
      "priority": "high",
      "title": "Bleeding after menopause",
      "body": "Any bleeding or spotting, even a little, 12 months or more after your last period should be checked. Usually the cause is simple, but an early check matters.",
      "action": "Tell your doctor about this",
      "link": "/menopause/alert",
      "needs_review": true,
      "data": {
        "last_on": "2026-09-22"
      }
    },
    {
      "key": "checkup_due",
      "kind": "reminder",
      "priority": "low",
      "title": "A checkup is due",
      "body": "It is time for Pap smear / HPV. Regular checks matter more for your bones and heart after menopause.",
      "action": "See my checkups",
      "link": "/checkups/3",
      "needs_review": true,
      "data": {
        "count": 2,
        "checkup": {
          "id": 3,
          "key": "pap_smear",
          "title": "Pap smear / HPV",
          "subtitle": "Cervical screening",
          "category": "multi_year",
          "section": "this_month",
          "status": "due",
          "icon": "shield",
          "tone": "violet",
          "interval_label": "Every 3 years",
          "timing_label": null,
          "last_done_on": null,
          "next_due_on": null,
          "next_due_label": "Due now",
          "is_custom": false
        }
      }
    },
    {
      "key": "stage_meno",
      "kind": "tip",
      "priority": "low",
      "title": "Menopause",
      "body": "It has been more than 12 months, which means you have reached menopause. From now on, log any bleeding or spotting and tell your doctor.",
      "action": null,
      "link": null,
      "needs_review": true,
      "data": null
    }
  ]
};
