package ir.ritmeapp.ritme.adapter.inbound.ui.log

import androidx.annotation.DrawableRes
import androidx.annotation.StringRes
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.domain.model.HealthLogCategory
import ir.ritmeapp.ritme.domain.model.HealthLogField

/**
 * The Persian copy for the health-log catalog (mirrors the web `log.json`). The domain stays
 * locale-free; this is the single UI-layer lookup from fields/categories/options to string
 * resources, so copy edits touch strings.xml only.
 */

@StringRes
internal fun HealthLogCategory.labelRes(): Int = when (this) {
    HealthLogCategory.BLEEDING -> R.string.log_cat_bleeding
    HealthLogCategory.PAIN -> R.string.log_cat_pain
    HealthLogCategory.DIGESTION -> R.string.log_cat_digestion
    HealthLogCategory.MOOD -> R.string.log_cat_mood
    HealthLogCategory.SLEEP -> R.string.log_cat_sleep
    HealthLogCategory.BODY -> R.string.log_cat_body
    HealthLogCategory.DISCHARGE -> R.string.log_cat_discharge
    HealthLogCategory.INTIMATE -> R.string.log_cat_intimate
    HealthLogCategory.SEXUAL -> R.string.log_cat_sexual
    HealthLogCategory.MEASURE -> R.string.log_cat_measure
    HealthLogCategory.NOTES -> R.string.log_cat_notes
}

@StringRes
internal fun HealthLogCategory.hintRes(): Int = when (this) {
    HealthLogCategory.BLEEDING -> R.string.log_hint_bleeding
    HealthLogCategory.PAIN -> R.string.log_hint_pain
    HealthLogCategory.DIGESTION -> R.string.log_hint_digestion
    HealthLogCategory.MOOD -> R.string.log_hint_mood
    HealthLogCategory.SLEEP -> R.string.log_hint_sleep
    HealthLogCategory.BODY -> R.string.log_hint_body
    HealthLogCategory.DISCHARGE -> R.string.log_hint_discharge
    HealthLogCategory.INTIMATE -> R.string.log_hint_intimate
    HealthLogCategory.SEXUAL -> R.string.log_hint_sexual
    HealthLogCategory.MEASURE -> R.string.log_hint_measure
    HealthLogCategory.NOTES -> R.string.log_hint_notes
}

@DrawableRes
internal fun HealthLogCategory.iconRes(): Int = when (this) {
    HealthLogCategory.BLEEDING -> R.drawable.ic_drop
    HealthLogCategory.PAIN -> R.drawable.ic_zap
    HealthLogCategory.DIGESTION -> R.drawable.ic_glass
    HealthLogCategory.MOOD -> R.drawable.ic_smile
    HealthLogCategory.SLEEP -> R.drawable.ic_moon
    HealthLogCategory.BODY -> R.drawable.ic_user
    HealthLogCategory.DISCHARGE -> R.drawable.ic_drop
    HealthLogCategory.INTIMATE -> R.drawable.ic_shield
    HealthLogCategory.SEXUAL -> R.drawable.ic_heart
    HealthLogCategory.MEASURE -> R.drawable.ic_thermo
    HealthLogCategory.NOTES -> R.drawable.ic_pencil
}

@StringRes
internal fun HealthLogField.labelRes(): Int = when (this) {
    HealthLogField.BLEEDING_INTENSITY -> R.string.log_field_bleeding_intensity
    HealthLogField.BLOOD_COLOR -> R.string.log_field_blood_color
    HealthLogField.BLEEDING_SMELL -> R.string.log_field_bleeding_smell
    HealthLogField.HAS_CLOTS -> R.string.log_field_has_clots
    HealthLogField.SPOTTING -> R.string.log_field_spotting
    HealthLogField.HEADACHE -> R.string.log_field_headache
    HealthLogField.STOMACH_ACHE -> R.string.log_field_stomach_ache
    HealthLogField.PELVIC_PAIN -> R.string.log_field_pelvic_pain
    HealthLogField.BREAST_PAIN -> R.string.log_field_breast_pain
    HealthLogField.BACK_PAIN -> R.string.log_field_back_pain
    HealthLogField.OVARIAN_PAIN -> R.string.log_field_ovarian_pain
    HealthLogField.NAUSEA -> R.string.log_field_nausea
    HealthLogField.BLOATING -> R.string.log_field_bloating
    HealthLogField.BREAST_SENSITIVITY -> R.string.log_field_breast_sensitivity
    HealthLogField.DIARRHEA -> R.string.log_field_diarrhea
    HealthLogField.CONSTIPATION -> R.string.log_field_constipation
    HealthLogField.APPETITE_CHANGE -> R.string.log_field_appetite_change
    HealthLogField.FOOD_CRAVING -> R.string.log_field_food_craving
    HealthLogField.MOODS -> R.string.log_field_moods
    HealthLogField.SLEEP_DURATION -> R.string.log_field_sleep_duration
    HealthLogField.SLEEP_QUALITY -> R.string.log_field_sleep_quality
    HealthLogField.ACNE -> R.string.log_field_acne
    HealthLogField.OILY_SKIN -> R.string.log_field_oily_skin
    HealthLogField.HAIR_LOSS -> R.string.log_field_hair_loss
    HealthLogField.SWELLING -> R.string.log_field_swelling
    HealthLogField.FATIGUE -> R.string.log_field_fatigue
    HealthLogField.DIZZINESS -> R.string.log_field_dizziness
    HealthLogField.HOT_FLASHES -> R.string.log_field_hot_flashes
    HealthLogField.CHILLS -> R.string.log_field_chills
    HealthLogField.DISCHARGE_TEXTURE -> R.string.log_field_discharge_texture
    HealthLogField.DISCHARGE_AMOUNT -> R.string.log_field_discharge_amount
    HealthLogField.DISCHARGE_SMELL -> R.string.log_field_discharge_smell
    HealthLogField.DISCHARGE_ITCHING -> R.string.log_field_discharge_itching
    HealthLogField.DISCHARGE_BURNING -> R.string.log_field_discharge_burning
    HealthLogField.FREQUENT_URINATION -> R.string.log_field_frequent_urination
    HealthLogField.URINATION_BURNING -> R.string.log_field_urination_burning
    HealthLogField.URINATION_CHANGE -> R.string.log_field_urination_change
    HealthLogField.VAGINAL_DRYNESS -> R.string.log_field_vaginal_dryness
    HealthLogField.VAGINAL_BURNING -> R.string.log_field_vaginal_burning
    HealthLogField.VAGINAL_ITCHING -> R.string.log_field_vaginal_itching
    HealthLogField.VAGINAL_SMELL_CHANGE -> R.string.log_field_vaginal_smell_change
    HealthLogField.SEXUAL_ACTIVITIES -> R.string.log_field_sexual_activities
    HealthLogField.WEIGHT -> R.string.log_field_weight
    HealthLogField.BASAL_BODY_TEMPERATURE -> R.string.log_field_bbt
    HealthLogField.NOTES -> R.string.log_field_notes
}

/** The Persian label for one enum option of [field]; falls back to the raw token if unknown. */
@StringRes
internal fun optionLabelRes(field: HealthLogField, option: String): Int? = when (field) {
    HealthLogField.BLEEDING_INTENSITY -> when (option) {
        "low" -> R.string.log_val_low
        "medium" -> R.string.log_val_medium
        "high" -> R.string.log_val_high
        "very_high" -> R.string.log_val_very_high
        else -> null
    }

    HealthLogField.BLOOD_COLOR -> when (option) {
        "bright_red" -> R.string.log_val_bright_red
        "red" -> R.string.log_val_red
        "dark_red" -> R.string.log_val_dark_red
        "brown" -> R.string.log_val_brown
        else -> null
    }

    HealthLogField.BLEEDING_SMELL, HealthLogField.DISCHARGE_SMELL -> when (option) {
        "normal" -> R.string.log_val_smell_normal
        "slightly_unusual" -> R.string.log_val_smell_slightly_unusual
        "strong_unpleasant" -> R.string.log_val_smell_strong_unpleasant
        else -> null
    }

    HealthLogField.APPETITE_CHANGE -> when (option) {
        "loss" -> R.string.log_val_appetite_loss
        "gain" -> R.string.log_val_appetite_gain
        "normal" -> R.string.log_val_appetite_normal
        else -> null
    }

    HealthLogField.URINATION_CHANGE -> when (option) {
        "increase" -> R.string.log_val_urination_increase
        "decrease" -> R.string.log_val_urination_decrease
        "normal" -> R.string.log_val_urination_normal
        else -> null
    }

    HealthLogField.MOODS -> when (option) {
        "happy" -> R.string.log_val_mood_happy
        "calm" -> R.string.log_val_mood_calm
        "angry" -> R.string.log_val_mood_angry
        "anxious" -> R.string.log_val_mood_anxious
        "sad" -> R.string.log_val_mood_sad
        "frustrated" -> R.string.log_val_mood_frustrated
        "sensitive" -> R.string.log_val_mood_sensitive
        "bored" -> R.string.log_val_mood_bored
        else -> null
    }

    HealthLogField.SLEEP_DURATION -> when (option) {
        "0_3" -> R.string.log_val_sleep_0_3
        "3_6" -> R.string.log_val_sleep_3_6
        "6_9" -> R.string.log_val_sleep_6_9
        "9_plus" -> R.string.log_val_sleep_9_plus
        else -> null
    }

    HealthLogField.SLEEP_QUALITY -> when (option) {
        "good" -> R.string.log_val_sleep_good
        "medium" -> R.string.log_val_sleep_medium
        "bad" -> R.string.log_val_sleep_bad
        else -> null
    }

    HealthLogField.SEXUAL_ACTIVITIES -> when (option) {
        "high_desire" -> R.string.log_val_sex_high_desire
        "protected_intercourse" -> R.string.log_val_sex_protected
        "unprotected_intercourse" -> R.string.log_val_sex_unprotected
        "no_desire" -> R.string.log_val_sex_no_desire
        "dryness" -> R.string.log_val_sex_dryness
        "burning" -> R.string.log_val_sex_burning
        "pain_during_intercourse" -> R.string.log_val_sex_pain
        "bleeding_after_intercourse" -> R.string.log_val_sex_bleeding_after
        "lubricant_use" -> R.string.log_val_sex_lubricant
        else -> null
    }

    HealthLogField.DISCHARGE_TEXTURE -> when (option) {
        "watery" -> R.string.log_val_discharge_watery
        "creamy" -> R.string.log_val_discharge_creamy
        "egg_white" -> R.string.log_val_discharge_egg_white
        "thick" -> R.string.log_val_discharge_thick
        else -> null
    }

    HealthLogField.DISCHARGE_AMOUNT -> when (option) {
        "low" -> R.string.log_val_low
        "medium" -> R.string.log_val_medium
        "high" -> R.string.log_val_high
        else -> null
    }

    // Every *_intensity degree field shares the same three-step scale.
    else -> when (option) {
        "low" -> R.string.log_val_low
        "medium" -> R.string.log_val_medium
        "high" -> R.string.log_val_high
        else -> null
    }
}
