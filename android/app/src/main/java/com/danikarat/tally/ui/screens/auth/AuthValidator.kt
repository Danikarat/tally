package com.danikarat.tally.ui.screens.auth

/**
 * AuthValidator provides client-side validation mirroring the backend constraints.
 */
object AuthValidator {

    private val EMAIL_REGEX = Regex("^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}$")

    fun validateEmail(email: String): String? {
        val trimmed = email.trim()
        if (trimmed.isBlank()) return "Email address cannot be empty"
        if (!EMAIL_REGEX.matches(trimmed)) return "Invalid email address format"
        return null
    }

    fun validatePassword(password: String): String? {
        if (password.length < 8) return "Password must be at least 8 characters long"
        if (!password.any { it.isUpperCase() }) return "Password must include at least one uppercase letter"
        if (!password.any { it.isLowerCase() }) return "Password must include at least one lowercase letter"
        if (!password.any { it.isDigit() }) return "Password must include at least one digit"
        return null
    }

    fun validateFullName(fullName: String): String? {
        val trimmed = fullName.trim()
        if (trimmed.length < 2) return "Full name must be at least 2 characters long"
        return null
    }
}
