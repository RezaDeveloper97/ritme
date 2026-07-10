package ir.ritmeapp.ritme.adapter.outbound.persistence

import android.content.ContentValues
import android.database.sqlite.SQLiteDatabase
import ir.ritmeapp.ritme.domain.model.SafeScreen
import ir.ritmeapp.ritme.domain.port.outbound.SafeStateRepository
import ir.ritmeapp.ritme.platform.log.RitmeLog
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * SQLite-backed [SafeStateRepository]. Best-effort by contract: any storage failure is
 * swallowed and logged rather than thrown, so recording the safe screen can never itself
 * crash the app (CLAUDE.md §7.2). All access is off the main thread via [ioDispatcher].
 */
class SqliteSafeStateRepository(
    private val helper: RitmeSqliteHelper,
    private val ioDispatcher: CoroutineDispatcher = Dispatchers.IO,
) : SafeStateRepository {

    override suspend fun save(screen: SafeScreen) {
        withContext(ioDispatcher) {
            try {
                val values = ContentValues().apply {
                    put(RitmeSqliteHelper.COL_ID, RitmeSqliteHelper.SINGLE_ROW_ID)
                    put(RitmeSqliteHelper.COL_ROUTE, screen.route)
                    put(RitmeSqliteHelper.COL_ARGS, screen.args)
                    put(RitmeSqliteHelper.COL_TIMESTAMP, screen.timestampMillis)
                }
                helper.writableDatabase.insertWithOnConflict(
                    RitmeSqliteHelper.TABLE_SAFE_SCREEN,
                    null,
                    values,
                    SQLiteDatabase.CONFLICT_REPLACE,
                )
            } catch (e: Exception) {
                RitmeLog.w(TAG, "Failed to persist safe screen", e)
            }
        }
    }

    override suspend fun lastSafeScreen(): SafeScreen? = withContext(ioDispatcher) {
        try {
            helper.readableDatabase.query(
                RitmeSqliteHelper.TABLE_SAFE_SCREEN,
                arrayOf(
                    RitmeSqliteHelper.COL_ROUTE,
                    RitmeSqliteHelper.COL_ARGS,
                    RitmeSqliteHelper.COL_TIMESTAMP,
                ),
                "${RitmeSqliteHelper.COL_ID} = ?",
                arrayOf(RitmeSqliteHelper.SINGLE_ROW_ID.toString()),
                null,
                null,
                null,
            ).use { cursor ->
                if (!cursor.moveToFirst()) return@use null
                SafeScreen(
                    route = cursor.getString(0),
                    args = if (cursor.isNull(1)) null else cursor.getString(1),
                    timestampMillis = cursor.getLong(2),
                )
            }
        } catch (e: Exception) {
            RitmeLog.w(TAG, "Failed to read safe screen", e)
            null
        }
    }

    private companion object {
        const val TAG = "SafeStateRepository"
    }
}
