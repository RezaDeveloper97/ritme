package ir.ritmeapp.ritme.adapter.outbound.network

import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpClient
import ir.ritmeapp.ritme.adapter.outbound.network.http.HttpMethod
import ir.ritmeapp.ritme.domain.model.AppError
import ir.ritmeapp.ritme.domain.model.AppResult
import ir.ritmeapp.ritme.domain.model.map
import ir.ritmeapp.ritme.domain.model.FetalMovementLog
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.model.PregnancyAlert
import ir.ritmeapp.ritme.domain.model.PregnancyAlertLevel
import ir.ritmeapp.ritme.domain.model.PregnancyConfidence
import ir.ritmeapp.ritme.domain.model.PregnancyOnboardingAnswers
import ir.ritmeapp.ritme.domain.model.PregnancyProfile
import ir.ritmeapp.ritme.domain.model.PregnancyStatus
import ir.ritmeapp.ritme.domain.model.PregnancySymptom
import ir.ritmeapp.ritme.domain.model.PregnancySymptomLog
import ir.ritmeapp.ritme.domain.model.PregnancyWeekContent
import ir.ritmeapp.ritme.domain.model.PregnancyWeeklyLog
import ir.ritmeapp.ritme.domain.model.SymptomSeverity
import ir.ritmeapp.ritme.domain.port.outbound.PregnancyGateway
import ir.ritmeapp.ritme.domain.port.outbound.TokenStore
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs
import org.json.JSONArray
import org.json.JSONObject

private const val HTTP_NOT_FOUND = 404

/**
 * Network [PregnancyGateway] over `/pregnancy/…`. Log reads treat a 404 as "nothing logged"
 * (a normal state), and a missing content week as empty content — the tracker never errors just
 * because a week's copy isn't seeded yet. Symptom values are never breadcrumbed (§11).
 */
class PregnancyGatewayAdapter(
    private val httpClient: HttpClient,
    private val tokenStore: TokenStore,
) : PregnancyGateway {

    override suspend fun activate(): AppResult<Unit> = confirm(HttpMethod.POST, ApiConfig.PREGNANCY_ACTIVATE_PATH)

    override suspend fun deactivate(): AppResult<Unit> = confirm(HttpMethod.POST, ApiConfig.PREGNANCY_DEACTIVATE_PATH)

    override suspend fun completeOnboarding(answers: PregnancyOnboardingAnswers): AppResult<Unit> =
        confirm(HttpMethod.POST, ApiConfig.PREGNANCY_ONBOARDING_PATH, serializeOnboarding(answers).toString())

    override suspend fun status(): AppResult<PregnancyStatus> {
        Breadcrumbs.add("api:pregnancy_status:start")
        return withData(HttpMethod.GET, ApiConfig.PREGNANCY_STATUS_PATH) { data ->
            val age = data.optJSONObject("gestational_age") ?: JSONObject()
            val due = data.optJSONObject("estimated_due_date")
            PregnancyStatus(
                isActive = data.optBoolean("is_active", false),
                gestationalWeeks = age.intOrNull("weeks"),
                gestationalDays = age.intOrNull("days"),
                trimester = age.intOrNull("trimester") ?: data.intOrNull("trimester"),
                dueDate = due?.stringOrNull("date")?.let(GregorianDate::parseIso),
                currentWeek = data.intOrNull("current_week"),
                isHighRisk = data.optBoolean("is_high_risk", false),
                fetalMovementTrackingActive = data.optBoolean("fetal_movement_tracking_active", false),
            )
        }
    }

    override suspend fun profile(): AppResult<PregnancyProfile?> {
        Breadcrumbs.add("api:pregnancy_profile:start")
        val result = httpClient.authorized(tokenStore, HttpMethod.GET, ApiConfig.PREGNANCY_PROFILE_PATH)
        return when (result) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                // No profile yet is a normal pre-onboarding state, not an error.
                if (result.value.statusCode == HTTP_NOT_FOUND) return AppResult.Success(null)
                when (val envelope = result.value.readEnvelope("Pregnancy profile rejected")) {
                    is AppResult.Failure -> envelope
                    is AppResult.Success -> try {
                        val profile = envelope.value.dataObject().optJSONObject("profile") ?: JSONObject()
                        AppResult.Success(
                            PregnancyProfile(
                                onboardingCompleted = profile.optBoolean("onboarding_completed", false),
                                confidenceLevel = PregnancyConfidence.fromApi(profile.stringOrNull("confidence_level")),
                                uncertaintyDays = profile.intOrNull("uncertainty_days"),
                                isLocked = profile.optBoolean("is_locked", false),
                            ),
                        )
                    } catch (e: Exception) {
                        AppResult.Failure(AppError.Parsing("Malformed pregnancy profile", e))
                    }
                }
            }
        }
    }

    override suspend fun contentForWeek(week: Int): AppResult<PregnancyWeekContent> {
        Breadcrumbs.add("api:pregnancy_content:w$week")
        val result = httpClient.authorized(tokenStore, HttpMethod.GET, ApiConfig.pregnancyContentPath(week))
        return when (result) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                // An unseeded week 404s; show the "not ready" empty state instead of an error.
                if (result.value.statusCode == HTTP_NOT_FOUND) return AppResult.Success(emptyContent(week))
                when (val envelope = result.value.readEnvelope("Pregnancy content rejected")) {
                    is AppResult.Failure -> envelope
                    is AppResult.Success -> try {
                        val content = envelope.value.dataObject().optJSONObject("content") ?: JSONObject()
                        AppResult.Success(
                            PregnancyWeekContent(
                                week = week,
                                fetalDevelopment = content.stringOrNull("fetal_development").orEmpty(),
                                motherBodyChanges = content.stringOrNull("mother_body_changes").orEmpty(),
                                bodyAdaptation = content.stringOrNull("body_adaptation").orEmpty(),
                                emotionalStatus = content.stringOrNull("emotional_status").orEmpty(),
                                keyNutrition = content.stringOrNull("key_nutrition").orEmpty(),
                                physicalActivity = content.stringOrNull("physical_activity").orEmpty(),
                                dosAndDonts = content.stringOrNull("dos_and_donts").orEmpty(),
                                carePlan = content.stringOrNull("care_plan").orEmpty(),
                                testsAndCheckups = content.stringOrNull("tests_and_checkups").orEmpty(),
                                faq = content.stringOrNull("faq").orEmpty(),
                            ),
                        )
                    } catch (e: Exception) {
                        AppResult.Failure(AppError.Parsing("Malformed pregnancy content", e))
                    }
                }
            }
        }
    }

    override suspend fun symptomLog(date: GregorianDate): AppResult<PregnancySymptomLog?> {
        Breadcrumbs.add("api:pregnancy_symptoms_get:start")
        val result = httpClient.authorized(tokenStore, HttpMethod.GET, ApiConfig.pregnancySymptomPath(date.toIso()))
        return when (result) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                if (result.value.statusCode == HTTP_NOT_FOUND) return AppResult.Success(null)
                when (val envelope = result.value.readEnvelope("Symptom log rejected")) {
                    is AppResult.Failure -> envelope
                    is AppResult.Success -> try {
                        val log = envelope.value.dataObject().optJSONObject("log") ?: JSONObject()
                        val symptoms = buildMap {
                            for (symptom in PregnancySymptom.entries) {
                                if (!log.optBoolean("has_${symptom.apiKey}", false)) continue
                                val severity = SymptomSeverity.fromApi(log.stringOrNull("${symptom.apiKey}_severity"))
                                put(symptom, severity ?: SymptomSeverity.MILD)
                            }
                        }
                        AppResult.Success(
                            PregnancySymptomLog(date, symptoms, log.stringOrNull("notes")),
                        )
                    } catch (e: Exception) {
                        AppResult.Failure(AppError.Parsing("Malformed symptom log", e))
                    }
                }
            }
        }
    }

    override suspend fun saveSymptomLog(log: PregnancySymptomLog): AppResult<Int> {
        Breadcrumbs.add("api:pregnancy_symptoms_save:start")
        val body = JSONObject().apply {
            put("log_date", log.date.toIso())
            log.symptoms.forEach { (symptom, severity) ->
                put("has_${symptom.apiKey}", true)
                put("${symptom.apiKey}_severity", severity.apiValue)
            }
            log.notes?.takeIf { it.isNotBlank() }?.let { put("notes", it) }
        }
        val result = httpClient.authorized(tokenStore, HttpMethod.POST, ApiConfig.PREGNANCY_SYMPTOMS_PATH, body.toString())
        return when (result) {
            is AppResult.Failure -> result
            is AppResult.Success -> result.value.readEnvelope("Symptom save rejected")
                .map { it.dataObject().optJSONArray("alerts")?.length() ?: 0 }
        }
    }

    override suspend fun weeklyLog(week: Int): AppResult<PregnancyWeeklyLog?> {
        Breadcrumbs.add("api:pregnancy_weekly_get:w$week")
        val result = httpClient.authorized(tokenStore, HttpMethod.GET, "${ApiConfig.PREGNANCY_WEEKLY_PATH}/$week")
        return when (result) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                if (result.value.statusCode == HTTP_NOT_FOUND) return AppResult.Success(null)
                when (val envelope = result.value.readEnvelope("Weekly log rejected")) {
                    is AppResult.Failure -> envelope
                    is AppResult.Success -> try {
                        val log = envelope.value.dataObject().optJSONObject("log") ?: JSONObject()
                        AppResult.Success(parseWeekly(week, log))
                    } catch (e: Exception) {
                        AppResult.Failure(AppError.Parsing("Malformed weekly log", e))
                    }
                }
            }
        }
    }

    override suspend fun saveWeeklyLog(log: PregnancyWeeklyLog): AppResult<Unit> {
        Breadcrumbs.add("api:pregnancy_weekly_save:start")
        return confirm(HttpMethod.POST, ApiConfig.PREGNANCY_WEEKLY_PATH, serializeWeekly(log).toString())
    }

    override suspend fun saveFetalMovement(log: FetalMovementLog): AppResult<Unit> {
        Breadcrumbs.add("api:pregnancy_movement_save:start")
        val body = JSONObject().apply {
            put("log_date", log.date.toIso())
            put("pregnancy_week", log.week)
            put("movement_status", log.movementStatus)
            log.movementCount?.let { put("movement_count", it) }
            log.firstMovementTime?.let { put("first_movement_time", it) }
            log.lastMovementTime?.let { put("last_movement_time", it) }
            log.notes?.takeIf { it.isNotBlank() }?.let { put("notes", it) }
        }
        return confirm(HttpMethod.POST, ApiConfig.PREGNANCY_FETAL_MOVEMENT_PATH, body.toString())
    }

    override suspend fun alerts(): AppResult<List<PregnancyAlert>> {
        Breadcrumbs.add("api:pregnancy_alerts:start")
        return withData(HttpMethod.GET, ApiConfig.PREGNANCY_ALERTS_PATH) { data ->
            val array = data.optJSONArray("alerts")
            (0 until (array?.length() ?: 0)).mapNotNull { index ->
                array?.optJSONObject(index)?.let { json ->
                    PregnancyAlert(
                        id = json.optLong("id"),
                        level = PregnancyAlertLevel.fromApi(json.stringOrNull("alert_level")),
                        title = json.stringOrNull("title").orEmpty(),
                        message = json.stringOrNull("message").orEmpty(),
                        recommendedActions = json.stringList("recommended_actions"),
                        isRead = json.optBoolean("is_read", false),
                    )
                }
            }
        }
    }

    override suspend fun markAlertRead(id: Long): AppResult<Unit> =
        confirm(HttpMethod.POST, ApiConfig.pregnancyAlertReadPath(id))

    override suspend fun dismissAlert(id: Long): AppResult<Unit> =
        confirm(HttpMethod.POST, ApiConfig.pregnancyAlertDismissPath(id))

    override suspend fun markAllAlertsRead(): AppResult<Unit> =
        confirm(HttpMethod.POST, ApiConfig.PREGNANCY_ALERTS_READ_ALL_PATH)

    // --- Shared plumbing ----------------------------------------------------

    private suspend fun confirm(method: HttpMethod, path: String, body: String? = null): AppResult<Unit> {
        val result = httpClient.authorized(tokenStore, method, path, body)
        return when (result) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                Breadcrumbs.add("api:pregnancy:http_${result.value.statusCode}")
                result.value.readEnvelope("Pregnancy request rejected").map { }
            }
        }
    }

    private suspend fun <T> withData(method: HttpMethod, path: String, read: (JSONObject) -> T): AppResult<T> {
        val result = httpClient.authorized(tokenStore, method, path)
        return when (result) {
            is AppResult.Failure -> result
            is AppResult.Success -> {
                Breadcrumbs.add("api:pregnancy:http_${result.value.statusCode}")
                when (val envelope = result.value.readEnvelope("Pregnancy request rejected")) {
                    is AppResult.Failure -> envelope
                    is AppResult.Success -> try {
                        AppResult.Success(read(envelope.value.dataObject()))
                    } catch (e: Exception) {
                        AppResult.Failure(AppError.Parsing("Malformed pregnancy response", e))
                    }
                }
            }
        }
    }

    private fun serializeOnboarding(answers: PregnancyOnboardingAnswers): JSONObject = JSONObject().apply {
        put("age_source", answers.ageSource.apiValue)
        answers.lmpDate?.let { put("lmp_date", it.toIso()) }
        answers.ultrasoundDate?.let { put("ultrasound_date", it.toIso()) }
        answers.ultrasoundWeeks?.let { put("ultrasound_weeks", it) }
        answers.ultrasoundDays?.let { put("ultrasound_days", it) }
        answers.manualWeeks?.let { put("manual_weeks", it) }
        answers.manualDays?.let { put("manual_days", it) }
        put("has_miscarriage_history", answers.hasMiscarriageHistory)
        put("has_high_risk_history", answers.hasHighRiskHistory)
        if (answers.preExistingConditions.isNotEmpty()) {
            put("pre_existing_conditions", JSONArray(answers.preExistingConditions))
        }
        answers.bloodType?.let { put("blood_type", it) }
        answers.rhFactor?.let { put("rh_factor", it) }
    }

    private fun parseWeekly(week: Int, log: JSONObject): PregnancyWeeklyLog = PregnancyWeeklyLog(
        week = week,
        date = log.stringOrNull("log_date")?.substringBefore('T')?.let(GregorianDate::parseIso)
            ?: GregorianDate(1970, 1, 1),
        weightKg = log.doubleOrNull("weight"),
        hasSwelling = log.optBoolean("has_swelling", false),
        swellingLocations = log.stringList("swelling_locations"),
        hasShortnessOfBreath = log.optBoolean("has_shortness_of_breath", false),
        hasBloodPressureDevice = log.optBoolean("has_blood_pressure_device", false),
        systolicPressure = log.intOrNull("systolic_pressure"),
        diastolicPressure = log.intOrNull("diastolic_pressure"),
        fastingBloodSugar = log.doubleOrNull("fasting_blood_sugar"),
        postMealBloodSugar = log.doubleOrNull("post_meal_blood_sugar"),
        overallMood = log.stringOrNull("overall_mood"),
        anxietySeverity = severityIf(log, "has_anxiety", "anxiety_severity"),
        moodSwingsSeverity = severityIf(log, "has_mood_swings", "mood_swings_severity"),
        depressionSeverity = severityIf(log, "has_depression_feelings", "depression_severity"),
        notes = log.stringOrNull("notes"),
    )

    private fun severityIf(log: JSONObject, hasKey: String, severityKey: String): SymptomSeverity? =
        if (log.optBoolean(hasKey, false)) {
            SymptomSeverity.fromApi(log.stringOrNull(severityKey)) ?: SymptomSeverity.MILD
        } else {
            null
        }

    private fun serializeWeekly(log: PregnancyWeeklyLog): JSONObject = JSONObject().apply {
        put("log_date", log.date.toIso())
        put("pregnancy_week", log.week)
        log.weightKg?.let { put("weight", it) }
        put("has_swelling", log.hasSwelling)
        if (log.swellingLocations.isNotEmpty()) put("swelling_locations", JSONArray(log.swellingLocations))
        put("has_shortness_of_breath", log.hasShortnessOfBreath)
        put("has_blood_pressure_device", log.hasBloodPressureDevice)
        log.systolicPressure?.let { put("systolic_pressure", it) }
        log.diastolicPressure?.let { put("diastolic_pressure", it) }
        log.fastingBloodSugar?.let { put("fasting_blood_sugar", it) }
        log.postMealBloodSugar?.let { put("post_meal_blood_sugar", it) }
        log.overallMood?.let { put("overall_mood", it) }
        log.anxietySeverity?.let {
            put("has_anxiety", true)
            put("anxiety_severity", it.apiValue)
        }
        log.moodSwingsSeverity?.let {
            put("has_mood_swings", true)
            put("mood_swings_severity", it.apiValue)
        }
        log.depressionSeverity?.let {
            put("has_depression_feelings", true)
            put("depression_severity", it.apiValue)
        }
        log.notes?.takeIf { it.isNotBlank() }?.let { put("notes", it) }
    }

    private fun emptyContent(week: Int): PregnancyWeekContent = PregnancyWeekContent(
        week = week,
        fetalDevelopment = "", motherBodyChanges = "", bodyAdaptation = "", emotionalStatus = "",
        keyNutrition = "", physicalActivity = "", dosAndDonts = "", carePlan = "",
        testsAndCheckups = "", faq = "",
    )
}
