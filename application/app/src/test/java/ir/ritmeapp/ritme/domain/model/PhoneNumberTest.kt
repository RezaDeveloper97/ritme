package ir.ritmeapp.ritme.domain.model

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PhoneNumberTest {

    private fun national(raw: String): String? =
        (PhoneNumber.parse(raw) as? AppResult.Success)?.value?.national

    @Test
    fun `accepts a standard 09 mobile number`() {
        assertEquals("09123456789", national("09123456789"))
    }

    @Test
    fun `normalizes the +98 country prefix`() {
        assertEquals("09123456789", national("+989123456789"))
    }

    @Test
    fun `normalizes the 0098 country prefix`() {
        assertEquals("09123456789", national("00989123456789"))
    }

    @Test
    fun `normalizes a bare 9-leading number`() {
        assertEquals("09123456789", national("9123456789"))
    }

    @Test
    fun `tolerates spaces and dashes`() {
        assertEquals("09123456789", national(" 0912-345 6789 "))
    }

    @Test
    fun `folds Persian digits to ASCII`() {
        assertEquals("09123456789", national("۰۹۱۲۳۴۵۶۷۸۹"))
    }

    @Test
    fun `exposes E164 form`() {
        val phone = (PhoneNumber.parse("09123456789") as AppResult.Success).value
        assertEquals("+989123456789", phone.e164)
    }

    @Test
    fun `rejects a too-short number`() {
        assertTrue(PhoneNumber.parse("0912345") is AppResult.Failure)
        assertFalse(PhoneNumber.isValid("0912345"))
    }

    @Test
    fun `rejects a landline number`() {
        assertTrue(PhoneNumber.parse("02112345678") is AppResult.Failure)
    }

    @Test
    fun `rejects empty input`() {
        assertTrue(PhoneNumber.parse("") is AppResult.Failure)
    }
}
