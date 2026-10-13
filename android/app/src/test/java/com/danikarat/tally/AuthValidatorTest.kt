package com.danikarat.tally

import com.danikarat.tally.ui.screens.auth.AuthValidator
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Test

class AuthValidatorTest {

    @Test
    fun validateEmail_acceptsValidEmails() {
        assertNull(AuthValidator.validateEmail("kwame.mensah@example.com"))
        assertNull(AuthValidator.validateEmail("user+gh@domain.co.gh"))
        assertNull(AuthValidator.validateEmail("dani@tally.money"))
    }

    @Test
    fun validateEmail_rejectsInvalidEmails() {
        assertNotNull(AuthValidator.validateEmail(""))
        assertNotNull(AuthValidator.validateEmail("   "))
        assertNotNull(AuthValidator.validateEmail("plainaddress"))
        assertNotNull(AuthValidator.validateEmail("@missinguser.com"))
        assertNotNull(AuthValidator.validateEmail("user@.com"))
        assertNotNull(AuthValidator.validateEmail("user@domain"))
    }

    @Test
    fun validatePassword_acceptsStrongPasswords() {
        assertNull(AuthValidator.validatePassword("StrongP@ssw0rd1"))
        assertNull(AuthValidator.validatePassword("P@ssword2026!"))
        assertNull(AuthValidator.validatePassword("GhanaAccra123"))
    }

    @Test
    fun validatePassword_rejectsWeakPasswords() {
        assertNotNull(AuthValidator.validatePassword("short1A")) // < 8 chars
        assertNotNull(AuthValidator.validatePassword("alllowercase1")) // missing uppercase
        assertNotNull(AuthValidator.validatePassword("ALLUPPERCASE1")) // missing lowercase
        assertNotNull(AuthValidator.validatePassword("NoDigitsPassword")) // missing digit
    }

    @Test
    fun validateFullName_acceptsValidNames() {
        assertNull(AuthValidator.validateFullName("Kwame Mensah"))
        assertNull(AuthValidator.validateFullName("Ab"))
    }

    @Test
    fun validateFullName_rejectsShortOrEmptyNames() {
        assertNotNull(AuthValidator.validateFullName(""))
        assertNotNull(AuthValidator.validateFullName(" "))
        assertNotNull(AuthValidator.validateFullName("A"))
    }
}
