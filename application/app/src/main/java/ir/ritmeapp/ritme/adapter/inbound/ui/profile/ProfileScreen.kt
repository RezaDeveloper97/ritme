package ir.ritmeapp.ritme.adapter.inbound.ui.profile

import android.content.Intent
import androidx.annotation.DrawableRes
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.SurfaceCard
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.formatFull
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.InfoTopic
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeBottomBar
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.RitmeTab
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.HealthProfile
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.domain.model.TrackingMode
import ir.ritmeapp.ritme.domain.model.UserProfile
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/**
 * The profile hub (web `/profile`): identity card, app-mode switch, the health snapshot, the
 * account/privacy/support row groups, logout, and the destructive delete-account flow behind a
 * confirmation sheet. Data comes from `GET /profile` + `GET /messages/mode`.
 */
@Composable
fun ProfileScreen(
    onNavigate: (Destination) -> Unit,
    onOpenSubScreen: (Destination) -> Unit,
    onSignedOut: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: ProfileViewModel = viewModel(factory = container.profileViewModelFactory())
    val state by viewModel.state.collectAsStateWithLifecycle()
    val colors = LocalRitmeColors.current
    val context = LocalContext.current

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:profile:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.Profile.route, null, System.currentTimeMillis()),
        )
        viewModel.effects.collect { effect ->
            when (effect) {
                ProfileEffect.SignedOut -> onSignedOut()
                is ProfileEffect.ShareExport -> {
                    val send = Intent(Intent.ACTION_SEND).apply {
                        type = "application/json"
                        putExtra(Intent.EXTRA_SUBJECT, "ritme-data-export.json")
                        putExtra(Intent.EXTRA_TEXT, effect.json)
                    }
                    context.startActivity(Intent.createChooser(send, null))
                }
            }
        }
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.background,
        bottomBar = { RitmeBottomBar(active = RitmeTab.PROFILE, mode = state.mode, onNavigate = onNavigate) },
    ) { padding ->
        if (state.loading && state.profile == null) {
            Box(Modifier.fillMaxSize().padding(padding), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = colors.pink)
            }
            return@Scaffold
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(padding),
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item(key = "title") {
                Text(
                    stringResource(R.string.profile_title),
                    style = MaterialTheme.typography.titleLarge,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.fillMaxWidth(),
                    textAlign = TextAlign.Center,
                )
            }
            item(key = "identity") {
                IdentityCard(state.profile, onEdit = { onOpenSubScreen(Destination.ProfilePersonal) }, colors)
            }
            item(key = "mode") { ModeGroup(state, viewModel::onIntent, onNavigate, colors) }
            item(key = "health") { HealthGroup(state.profile, colors) }
            state.profile?.bmi?.message?.takeIf { it.isNotBlank() }?.let { bmiMessage ->
                item(key = "bmi") {
                    SurfaceCard {
                        Text(bmiMessage, style = MaterialTheme.typography.bodyMedium, color = colors.inkMuted)
                    }
                }
            }
            item(key = "account") {
                RowGroup(stringResource(R.string.profile_section_account), colors) {
                    NavRow(R.drawable.ic_user, stringResource(R.string.profile_row_personal), colors) {
                        onOpenSubScreen(Destination.ProfilePersonal)
                    }
                    NavRow(R.drawable.ic_refresh, stringResource(R.string.profile_row_health), colors) {
                        onOpenSubScreen(Destination.ProfileHealth)
                    }
                    NavRow(R.drawable.ic_alarm, stringResource(R.string.profile_row_reminders), colors) {
                        onOpenSubScreen(Destination.ProfileReminders)
                    }
                    NavRow(R.drawable.ic_bell, stringResource(R.string.profile_row_notifications), colors) {
                        onOpenSubScreen(Destination.ProfileNotifications)
                    }
                }
            }
            item(key = "privacy") {
                RowGroup(stringResource(R.string.profile_section_privacy), colors) {
                    NavRow(R.drawable.ic_shield, stringResource(R.string.profile_row_privacy), colors) {
                        onOpenSubScreen(Destination.ProfileInfo(InfoTopic.PRIVACY))
                    }
                    NavRow(
                        icon = R.drawable.ic_download,
                        label = if (state.exporting) {
                            stringResource(R.string.profile_exporting)
                        } else {
                            stringResource(R.string.profile_row_export)
                        },
                        colors = colors,
                    ) { viewModel.onIntent(ProfileIntent.Export) }
                    if (state.exportError) {
                        Text(
                            stringResource(R.string.profile_export_error),
                            style = MaterialTheme.typography.labelSmall,
                            color = colors.error,
                            modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp),
                        )
                    }
                    NavRow(
                        icon = R.drawable.ic_trash,
                        label = stringResource(R.string.profile_row_delete),
                        colors = colors,
                        danger = true,
                    ) { viewModel.onIntent(ProfileIntent.OpenDeleteSheet) }
                }
            }
            item(key = "support") {
                RowGroup(stringResource(R.string.profile_section_support), colors) {
                    NavRow(R.drawable.ic_info, stringResource(R.string.profile_row_help), colors) {
                        onOpenSubScreen(Destination.ProfileInfo(InfoTopic.HELP))
                    }
                    NavRow(R.drawable.ic_heart, stringResource(R.string.profile_row_about), colors) {
                        onOpenSubScreen(Destination.ProfileInfo(InfoTopic.ABOUT))
                    }
                    NavRow(R.drawable.ic_book_open, stringResource(R.string.profile_row_terms), colors) {
                        onOpenSubScreen(Destination.ProfileInfo(InfoTopic.TERMS))
                    }
                }
            }
            item(key = "logout") {
                SurfaceCard(Modifier.clickable { viewModel.onIntent(ProfileIntent.Logout) }) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(
                            painter = painterResource(R.drawable.ic_logout),
                            contentDescription = null,
                            tint = colors.error,
                            modifier = Modifier.size(20.dp),
                        )
                        Spacer(Modifier.width(10.dp))
                        Text(
                            text = if (state.loggingOut) {
                                stringResource(R.string.profile_logging_out)
                            } else {
                                stringResource(R.string.profile_logout)
                            },
                            style = MaterialTheme.typography.bodyMedium,
                            color = colors.error,
                            fontWeight = FontWeight.Bold,
                        )
                    }
                }
            }
            item(key = "version") {
                Text(
                    text = stringResource(R.string.profile_version, "۱٫۰٫۰"),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.inkMuted,
                    modifier = Modifier.fillMaxWidth(),
                    textAlign = TextAlign.Center,
                )
            }
            item(key = "tail") { Spacer(Modifier.height(4.dp)) }
        }
    }

    if (state.deleteSheetOpen) {
        DeleteAccountSheet(state, viewModel::onIntent, colors)
    }
}

@Composable
private fun IdentityCard(profile: UserProfile?, onEdit: () -> Unit, colors: RitmeColors) {
    SurfaceCard {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(
                Modifier.size(52.dp).clip(CircleShape).background(colors.pinkContainer),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_user),
                    contentDescription = null,
                    tint = colors.pink,
                    modifier = Modifier.size(24.dp),
                )
            }
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    text = profile?.account?.name?.takeIf { it.isNotBlank() }
                        ?: stringResource(R.string.profile_guest),
                    style = MaterialTheme.typography.titleMedium,
                    color = colors.ink,
                    fontWeight = FontWeight.Bold,
                )
                Text(
                    text = profile?.account?.mobile?.takeIf { it.isNotBlank() }?.toPersianDigits()
                        ?: stringResource(R.string.profile_no_phone),
                    style = MaterialTheme.typography.labelMedium,
                    color = colors.inkMuted,
                )
            }
            Box(
                Modifier.size(38.dp).clip(CircleShape).background(colors.background).clickable(onClick = onEdit),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_pencil),
                    contentDescription = stringResource(R.string.action_edit),
                    tint = colors.pink,
                    modifier = Modifier.size(18.dp),
                )
            }
        }
    }
}

@Composable
private fun ModeGroup(
    state: ProfileUiState,
    onIntent: (ProfileIntent) -> Unit,
    onNavigate: (Destination) -> Unit,
    colors: RitmeColors,
) {
    RowGroup(stringResource(R.string.profile_section_mode), colors) {
        if (state.mode == TrackingMode.PREGNANCY) {
            NavRow(R.drawable.ic_heart, stringResource(R.string.profile_mode_tracker), colors) {
                onNavigate(Destination.Pregnancy)
            }
            NavRow(
                icon = R.drawable.ic_refresh,
                label = if (state.switchingMode) {
                    stringResource(R.string.profile_mode_switching)
                } else {
                    stringResource(R.string.profile_mode_to_cycle)
                },
                colors = colors,
            ) { onIntent(ProfileIntent.SwitchToCycleMode) }
        } else {
            NavRow(R.drawable.ic_heart, stringResource(R.string.profile_mode_to_pregnancy), colors) {
                onNavigate(Destination.Pregnancy)
            }
        }
    }
}

@Composable
private fun HealthGroup(profile: UserProfile?, colors: RitmeColors) {
    val health = profile?.health
    RowGroup(stringResource(R.string.profile_section_health), colors) {
        if (health == null) {
            Text(
                stringResource(R.string.profile_health_not_set_up),
                style = MaterialTheme.typography.labelMedium,
                color = colors.inkMuted,
                modifier = Modifier.padding(16.dp),
            )
            return@RowGroup
        }
        ValueRow(stringResource(R.string.profile_health_cycle), daysValue(health.cycleDuration), colors)
        ValueRow(stringResource(R.string.profile_health_period), daysValue(health.periodDuration), colors)
        ValueRow(
            label = stringResource(R.string.profile_health_last_period),
            value = health.lastPeriodStart?.toJalali()?.formatFull() ?: stringResource(R.string.value_empty),
            colors = colors,
        )
        ValueRow(
            label = stringResource(R.string.profile_health_birthday),
            value = health.birthday?.toJalali()?.formatFull() ?: stringResource(R.string.value_empty),
            colors = colors,
        )
        ValueRow(stringResource(R.string.profile_health_weight), weightValue(health), colors)
        ValueRow(stringResource(R.string.profile_health_height), heightValue(health), colors)
        profile.bmi?.let { bmi ->
            ValueRow(
                label = stringResource(R.string.profile_health_bmi),
                value = "${formatDecimal(bmi.value)} · ${bmi.categoryLabel.ifBlank { bmi.category }}",
                colors = colors,
            )
        }
    }
}

@Composable
private fun daysValue(days: Int?): String = days
    ?.let { stringResource(R.string.profile_health_days, it.toPersianDigits()) }
    ?: stringResource(R.string.value_empty)

@Composable
private fun weightValue(health: HealthProfile): String = health.weightKg
    ?.let { stringResource(R.string.profile_health_kg, formatDecimal(it)) }
    ?: stringResource(R.string.value_empty)

@Composable
private fun heightValue(health: HealthProfile): String = health.heightCm
    ?.let { stringResource(R.string.profile_health_cm, it.toPersianDigits()) }
    ?: stringResource(R.string.value_empty)

/** «۶۲٫۵» — drops a whole number's decimal and localizes digits. */
private fun formatDecimal(value: Double): String {
    val text = if (value % 1.0 == 0.0) value.toInt().toString() else String.format(java.util.Locale.US, "%.1f", value)
    return text.toPersianDigits()
}

/** A titled white group of rows (web's grouped list cards). */
@Composable
private fun RowGroup(title: String, colors: RitmeColors, content: @Composable () -> Unit) {
    Column {
        Text(
            text = title,
            style = MaterialTheme.typography.labelMedium,
            color = colors.inkMuted,
            fontWeight = FontWeight.Bold,
            modifier = Modifier.padding(horizontal = 6.dp, vertical = 6.dp),
        )
        Column(
            Modifier
                .fillMaxWidth()
                .clip(RoundedCornerShape(16.dp))
                .background(colors.surface),
        ) {
            content()
        }
    }
}

@Composable
private fun NavRow(
    @DrawableRes icon: Int,
    label: String,
    colors: RitmeColors,
    danger: Boolean = false,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier.fillMaxWidth().clickable(onClick = onClick).padding(horizontal = 14.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            Modifier.size(34.dp).clip(CircleShape)
                .background(if (danger) colors.periodContainer else colors.background),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(icon),
                contentDescription = null,
                tint = if (danger) colors.error else colors.inkMuted,
                modifier = Modifier.size(17.dp),
            )
        }
        Spacer(Modifier.width(12.dp))
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = if (danger) colors.error else colors.ink,
            modifier = Modifier.weight(1f),
        )
        Icon(
            // RTL: the forward affordance points toward the text end.
            painter = painterResource(R.drawable.ic_chevron_left),
            contentDescription = null,
            tint = colors.outline,
            modifier = Modifier.size(16.dp),
        )
    }
}

@Composable
private fun ValueRow(label: String, value: String, colors: RitmeColors) {
    Row(
        modifier = Modifier.fillMaxWidth().padding(horizontal = 14.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(label, style = MaterialTheme.typography.bodyMedium, color = colors.inkMuted)
        Text(value, style = MaterialTheme.typography.bodyMedium, color = colors.ink, fontWeight = FontWeight.Bold)
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun DeleteAccountSheet(state: ProfileUiState, onIntent: (ProfileIntent) -> Unit, colors: RitmeColors) {
    ModalBottomSheet(
        onDismissRequest = { onIntent(ProfileIntent.CloseDeleteSheet) },
        containerColor = colors.surface,
    ) {
        Column(
            Modifier.fillMaxWidth().padding(horizontal = 20.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Box(
                Modifier.size(52.dp).clip(CircleShape).background(colors.periodContainer),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    painter = painterResource(R.drawable.ic_trash),
                    contentDescription = null,
                    tint = colors.error,
                    modifier = Modifier.size(24.dp),
                )
            }
            Spacer(Modifier.height(12.dp))
            Text(
                stringResource(R.string.delete_title),
                style = MaterialTheme.typography.titleMedium,
                color = colors.ink,
                fontWeight = FontWeight.Bold,
            )
            Spacer(Modifier.height(8.dp))
            Text(
                stringResource(R.string.delete_warning),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.inkMuted,
                textAlign = TextAlign.Center,
            )
            if (state.deleteError) {
                Spacer(Modifier.height(8.dp))
                Text(
                    stringResource(R.string.delete_error),
                    style = MaterialTheme.typography.labelSmall,
                    color = colors.error,
                )
            }
            Spacer(Modifier.height(16.dp))
            Button(
                onClick = { onIntent(ProfileIntent.ConfirmDelete) },
                enabled = !state.deleting,
                colors = ButtonDefaults.buttonColors(containerColor = colors.error, contentColor = colors.onPink),
                shape = RoundedCornerShape(14.dp),
                modifier = Modifier.fillMaxWidth().height(46.dp),
            ) {
                Text(
                    text = if (state.deleting) {
                        stringResource(R.string.delete_deleting)
                    } else {
                        stringResource(R.string.delete_confirm)
                    },
                    fontWeight = FontWeight.Bold,
                )
            }
            Spacer(Modifier.height(8.dp))
            Text(
                text = stringResource(R.string.action_cancel),
                style = MaterialTheme.typography.bodyMedium,
                color = colors.inkMuted,
                modifier = Modifier
                    .clip(RoundedCornerShape(12.dp))
                    .clickable { onIntent(ProfileIntent.CloseDeleteSheet) }
                    .padding(10.dp),
            )
            Spacer(Modifier.height(16.dp))
        }
    }
}
