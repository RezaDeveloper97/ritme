package ir.ritmeapp.ritme.adapter.inbound.ui.navigation

import androidx.annotation.DrawableRes
import androidx.annotation.StringRes
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import ir.ritmeapp.ritme.R
import ir.ritmeapp.ritme.adapter.inbound.ui.theme.LocalRitmeColors
import ir.ritmeapp.ritme.domain.model.TrackingMode

/** The five slots of the tab bar; LOG is the raised center action. */
enum class RitmeTab { TODAY, CALENDAR, LOG, CYCLE, PREGNANCY, PROFILE }

/**
 * The floating pill tab bar shared by every root tab (web `.tabbar`): white rounded surface,
 * a raised gradient FAB in the middle, and a gradient underline on the active tab. Mode-aware:
 * in pregnancy mode the چرخه tab becomes بارداری and the FAB opens the pregnancy log.
 * Hand-built, no navigation library (§3).
 */
@Composable
fun RitmeBottomBar(
    active: RitmeTab,
    mode: TrackingMode,
    onNavigate: (Destination) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalRitmeColors.current
    val trackerTab = if (mode == TrackingMode.PREGNANCY) RitmeTab.PREGNANCY else RitmeTab.CYCLE
    Row(
        modifier = modifier
            .fillMaxWidth()
            .padding(horizontal = 12.dp, vertical = 10.dp)
            .shadow(12.dp, RoundedCornerShape(36.dp))
            .clip(RoundedCornerShape(36.dp))
            .background(colors.surface)
            .padding(horizontal = 6.dp, vertical = 8.dp),
        horizontalArrangement = Arrangement.SpaceEvenly,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        TabItem(R.drawable.ic_home, R.string.nav_today, active == RitmeTab.TODAY) {
            onNavigate(Destination.Home)
        }
        TabItem(R.drawable.ic_calendar, R.string.nav_calendar, active == RitmeTab.CALENDAR) {
            onNavigate(Destination.Calendar)
        }
        LogFab {
            onNavigate(
                if (mode == TrackingMode.PREGNANCY) Destination.PregnancyLog() else Destination.Log(),
            )
        }
        if (trackerTab == RitmeTab.PREGNANCY) {
            TabItem(R.drawable.ic_heart, R.string.nav_pregnancy, active == RitmeTab.PREGNANCY) {
                onNavigate(Destination.Pregnancy)
            }
        } else {
            TabItem(R.drawable.ic_refresh, R.string.nav_cycle, active == RitmeTab.CYCLE) {
                onNavigate(Destination.Cycle)
            }
        }
        TabItem(R.drawable.ic_user, R.string.nav_profile, active == RitmeTab.PROFILE) {
            onNavigate(Destination.Profile)
        }
    }
}

@Composable
private fun TabItem(@DrawableRes icon: Int, @StringRes label: Int, isActive: Boolean, onClick: () -> Unit) {
    val colors = LocalRitmeColors.current
    val activeBrush = Brush.linearGradient(listOf(colors.pinkContainer, colors.background))
    Column(
        modifier = Modifier
            .clip(RoundedCornerShape(14.dp))
            .then(if (isActive) Modifier.background(activeBrush) else Modifier)
            .clickable(
                interactionSource = remember { MutableInteractionSource() },
                indication = null,
                onClick = onClick,
            )
            .padding(horizontal = 12.dp, vertical = 6.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(
            painter = painterResource(icon),
            contentDescription = null,
            tint = if (isActive) colors.pink else colors.inkMuted,
            modifier = Modifier.size(22.dp),
        )
        Spacer(Modifier.height(3.dp))
        Text(
            text = stringResource(label),
            style = MaterialTheme.typography.labelSmall,
            color = if (isActive) colors.pinkDark else colors.inkMuted,
            fontWeight = if (isActive) FontWeight.ExtraBold else FontWeight.Normal,
        )
        Spacer(Modifier.height(2.dp))
        Box(
            Modifier
                .width(26.dp)
                .height(3.dp)
                .clip(RoundedCornerShape(99.dp))
                .background(
                    if (isActive) Brush.linearGradient(listOf(colors.pink, colors.accent))
                    else Brush.linearGradient(listOf(colors.surface, colors.surface)),
                ),
        )
    }
}

@Composable
private fun LogFab(onClick: () -> Unit) {
    val colors = LocalRitmeColors.current
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Box(
            modifier = Modifier
                .offset(y = (-10).dp)
                .size(52.dp)
                .shadow(8.dp, CircleShape)
                .clip(CircleShape)
                .background(Brush.verticalGradient(listOf(colors.pink, colors.accent)))
                .clickable(onClick = onClick),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                painter = painterResource(R.drawable.ic_plus),
                contentDescription = stringResource(R.string.nav_log),
                tint = colors.onPink,
                modifier = Modifier.size(24.dp),
            )
        }
        Text(
            text = stringResource(R.string.nav_log),
            style = MaterialTheme.typography.labelSmall,
            color = colors.inkMuted,
            modifier = Modifier.offset(y = (-8).dp),
        )
    }
}
