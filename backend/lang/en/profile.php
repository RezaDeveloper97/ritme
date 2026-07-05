<?php

return [
    'unauthenticated' => 'Unauthenticated. Please provide a valid Bearer token.',
    'unknown_fields' => 'Unknown field(s) in request: :fields',
    'validation_failed' => 'Validation failed: :first',
    'updated' => 'Profile updated successfully',
    'db_error' => 'Database error while saving profile.',
    'unexpected_error' => 'An unexpected error occurred while updating your profile.',
    'try_again' => 'Please try again later.',

    'errors' => [
        'name_string' => 'Name must be a string.',
        'name_max' => 'Name must not exceed 255 characters.',
        'birthday_date' => 'Birthday must be a valid date.',
        'birthday_before' => 'Birthday must be a date before today.',
        'weight_numeric' => 'Weight must be a number.',
        'weight_min' => 'Weight must be at least 20 kg.',
        'weight_max' => 'Weight must not exceed 300 kg.',
        'height_integer' => 'Height must be an integer.',
        'height_min' => 'Height must be at least 50 cm.',
        'height_max' => 'Height must not exceed 250 cm.',
        'period_duration_integer' => 'Period duration must be an integer.',
        'period_duration_min' => 'Period duration must be at least 1 day.',
        'period_duration_max' => 'Period duration must not exceed 15 days.',
        'cycle_duration_integer' => 'Cycle duration must be an integer.',
        'cycle_duration_min' => 'Cycle duration must be at least 15 days.',
        'cycle_duration_max' => 'Cycle duration must not exceed 60 days.',
        'last_period_start_date' => 'Last period start must be a valid date.',
        'last_period_start_before_or_equal' => 'Last period start cannot be in the future.',
        'user_goal_in' => 'user_goal must be one of: :values',
        'subscription_type_in' => 'subscription_type must be one of: :values',
    ],
];
