package ir.ritmeapp.ritme.adapter.inbound.ui.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.ScreenHeader
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.formatFull
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.toPersianDigits
import ir.ritmeapp.ritme.adapter.inbound.ui.foundation.todayJalali
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.Destination
import ir.ritmeapp.ritme.adapter.inbound.ui.navigation.LocalAppContainer
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.RitmeColors
import ir.ritmeapp.ritme.domain.model.AppNotification
import ir.ritmeapp.ritme.domain.model.GregorianDate
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.platform.crash.Breadcrumbs

/** Web `--pink-soft` (#FFCFCF): the unread-row wash and empty-state badge tint. */
private val PinkSoft = Color(0xFFFFCFCF) // TODO token (--pink-soft)

/**
 * The notifications inbox (web `/profile/notifications`): one bordered card of divider-separated
 * rows, unread rows tinted pink with a filled status dot; tapping an unread row marks it read,
 * «خواندن همه» clears everything.
 */
@Composable
fun NotificationsScreen(
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val container = LocalAppContainer.current
    val viewModel: NotificationsViewModel = viewModel(factory = container.notificationsViewModelFactory())
    val state by viewModel.state.collectAsStateWithLifecycle()
    val colors = LocalRitmeColors.current

    LaunchedEffect(Unit) {
        Breadcrumbs.add("screen:notifications:enter")
        container.safeScreenTracker.record(
            SafeScreen(Destination.ProfileNotifications.route, null, System.currentTimeMillis()),
        )
    }

    // Newest first regardless of backend ordering (ISO strings sort lexically, like the web).
    val items = state.page?.items.orEmpty().sortedByDescending { it.createdAt }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        containerColor = colors.background,
        topBar = {
            ScreenHeader(
                title = stringResource(R.string.notifications_title),
                onBack = onBack,
                trailing = {
                    if ((state.page?.unreadCount ?: 0) > 0) {
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(5.dp),
                            modifier = Modifier
                                .clip(RoundedCornerShape(10.dp))
                                .clickable { viewModel.onIntent(NotificationsIntent.MarkAllRead) }
                                .padding(horizontal = 8.dp, vertical = 6.dp),
                        ) {
                            Icon(
                                painter = painterResource(R.drawable.ic_check_circle),
                                contentDescription = null,
                                tint = colors.pink,
                                modifier = Modifier.size(16.dp),
                            )
                            Text(
                                text = stringResource(R.string.notifications_mark_all),
                                style = MaterialTheme.typography.labelMedium,
                                color = colors.pink,
                                fontWeight = FontWeight.Bold,
                            )
                        }
                    }
                },
            )
        },
    ) { padding ->
        when {
            // Web renders nothing while the query is pending and there are no items yet.
            items.isEmpty() && state.loading ->
                Box(Modifier.fillMaxSize().padding(padding))

            items.isEmpty() -> EmptyInbox(colors, Modifier.padding(padding))

            else -> LazyColumn(
                modifier = Modifier.fillMaxSize().padding(padding),
                contentPadding = PaddingValues(start = 18.dp, end = 18.dp, top = 8.dp, bottom = 24.dp),
            ) {
                item(key = "card") {
                    Column(
                        Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(16.dp))
                            .background(colors.surface)
                            .border(1.dp, colors.outline, RoundedCornerShape(16.dp)),
                    ) {
                        items.forEachIndexed { index, notification ->
                            if (index > 0) {
                                Box(
                                    Modifier
                                        .fillMaxWidth()
                                        .padding(start = 35.dp)
                                        .height(1.dp)
                                        .background(colors.outline),
                                )
                            }
                            NotificationRow(
                                notification = notification,
                                onTap = { viewModel.onIntent(NotificationsIntent.MarkRead(notification.id)) },
                                colors = colors,
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun EmptyInbox(colors: RitmeColors, modifier: Modifier = Modifier) {
    Column(
        modifier
            .fillMaxSize()
            .padding(start = 32.dp, end = 32.dp, top = 72.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier
                .size(64.dp)
                .clip(CircleShape)
                .background(PinkSoft),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_bell),
                contentDescription = null,
                tint = colors.pink,
                modifier = Modifier.size(30.dp),
            )
        }
        Spacer(Modifier.height(8.dp))
        Text(
            stringResource(R.string.notifications_empty_title),
            style = MaterialTheme.typography.titleMedium.copy(fontSize = 15.sp),
            color = colors.ink,
            fontWeight = FontWeight.ExtraBold,
        )
        Spacer(Modifier.height(6.dp))
        Text(
            stringResource(R.string.notifications_empty_body),
            style = MaterialTheme.typography.labelMedium.copy(fontSize = 13.sp, lineHeight = 23.sp),
            color = colors.inkMuted,
            textAlign = TextAlign.Center,
        )
    }
}

@Composable
private fun NotificationRow(notification: AppNotification, onTap: () -> Unit, colors: RitmeColors) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(if (notification.isRead) Color.Transparent else PinkSoft)
            .clickable(enabled = !notification.isRead, onClick = onTap)
            .padding(horizontal = 14.dp, vertical = 13.dp),
        verticalAlignment = Alignment.Top,
    ) {
        // Status dot on every row: filled brand when unread, an outlined placeholder when read,
        // so titles stay aligned across both states.
        Box(
            Modifier
                .padding(top = 6.dp)
                .size(9.dp)
                .clip(CircleShape)
                .then(
                    if (notification.isRead) {
                        Modifier.border(1.dp, colors.outline, CircleShape)
                    } else {
                        Modifier.background(colors.pink)
                    },
                ),
        )
        Spacer(Modifier.width(12.dp))
        Column(Modifier.weight(1f)) {
            Text(
                text = notification.title,
                style = MaterialTheme.typography.bodyMedium.copy(fontSize = 14.sp),
                color = colors.ink,
                fontWeight = if (notification.isRead) FontWeight.SemiBold else FontWeight.ExtraBold,
            )
            if (notification.body.isNotBlank()) {
                Spacer(Modifier.height(3.dp))
                Text(
                    text = notification.body,
                    style = MaterialTheme.typography.labelMedium.copy(fontSize = 13.sp, lineHeight = 22.sp),
                    color = colors.inkMuted,
                )
            }
            Spacer(Modifier.height(6.dp))
            Text(
                text = relativeTimeLabel(notification.createdAt),
                style = MaterialTheme.typography.labelSmall.copy(fontSize = 11.sp),
                color = colors.inkMuted,
            )
        }
    }
}

/** «امروز» / «دیروز» / «n روز پیش» / a full Jalali date past a week — like the web inbox. */
@Composable
private fun relativeTimeLabel(createdAtIso: String): String {
    val date = remember(createdAtIso) {
        createdAtIso.substringBefore('T').substringBefore(' ').let(GregorianDate::parseIso)?.toJalali()
    } ?: return ""
    val daysAgo = todayJalali().toJdn() - date.toJdn()
    return when {
        daysAgo <= 0 -> stringResource(R.string.notifications_today)
        daysAgo == 1 -> stringResource(R.string.notifications_yesterday)
        daysAgo <= DAYS_IN_WEEK -> stringResource(R.string.notifications_days_ago, daysAgo.toPersianDigits())
        else -> date.formatFull()
    }
}

private const val DAYS_IN_WEEK = 7
