<?php

return [
    // Phases
    'phases' => [
        'menstruation' => 'Menstruation',
        'follicular' => 'Follicular',
        'ovulation' => 'Ovulation',
        'luteal' => 'Luteal',
    ],

    'phase_descriptions' => [
        'menstruation' => 'Monthly bleeding period',
        'follicular' => 'Follicle growth and egg preparation phase',
        'ovulation' => 'Egg release period',
        'luteal' => 'Post-ovulation phase',
    ],

    // Subphases
    'subphases' => [
        'menstruation' => 'Menstruation',
        'early_follicular' => 'Early Follicular',
        'late_follicular' => 'Late Follicular',
        'ovulation_window' => 'Ovulation Window',
        'early_luteal' => 'Early Luteal',
        'mid_luteal' => 'Mid Luteal',
        'late_luteal' => 'Late Luteal (PMS)',
    ],

    // Variability
    'variability' => [
        'regular' => 'Regular',
        'semi_irregular' => 'Semi-Irregular',
        'irregular' => 'Irregular',
    ],

    // Status
    'status' => [
        'pending' => 'Pending',
        'processing' => 'Processing',
        'completed' => 'Completed',
        'failed' => 'Failed',
    ],

    // Messages
    'messages' => [
        'fertility_high' => 'You are in your fertile window. Pregnancy chance is higher.',
        'probability' => "Today's pregnancy probability is :percent%.",
        'pms_warning' => 'You are in the PMS window. You may experience mood changes and physical symptoms.',
        'period_prediction' => 'Your period may start soon (±:days days based on your cycle regularity).',
        'phase_info' => 'Current phase: :phase (:subphase)',
        'incomplete_profile' => 'Please complete your profile to get cycle predictions.',
        'recalculation_started' => 'Recalculation started',
        'already_processing' => 'Calculation already in progress',
        'complete_profile_first' => 'Please complete your profile first',
    ],

    // Tips
    'tips' => [
        'hydration' => 'Stay hydrated and drink plenty of water.',
        'warmth' => 'Apply a warm compress to your lower abdomen to relieve cramps.',
        'rest' => 'Get enough rest and avoid strenuous activities.',
        'iron_foods' => 'Eat iron-rich foods like spinach and red meat.',
        'energy_rising' => 'Your energy levels are rising. Great time for new projects!',
        'high_intensity_workout' => 'Perfect time for high-intensity workouts.',
        'peak_fertility' => 'Peak fertility days. If trying to conceive, this is the best time.',
        'confidence' => 'You may feel more confident and social.',
        'pms_selfcare' => 'PMS symptoms may appear. Practice self-care and relaxation.',
        'reduce_salt' => 'Reduce salt and caffeine intake to minimize bloating.',
        'deep_breathing' => 'Take breaks and practice deep breathing if feeling irritable.',
        'magnesium_b6' => 'Focus on foods rich in magnesium and vitamin B6.',
        'headache_relief' => 'For headaches: rest in a dark room and stay hydrated.',
        'cramp_relief' => 'For cramps: try a warm bath or heating pad.',
        'sleep_schedule' => 'Try to maintain a regular sleep schedule and avoid screens before bed.',
        'mental_health' => 'Take time for activities you enjoy. Consider light exercise or meditation.',
        'bloating_relief' => 'Reduce salt intake and eat smaller, frequent meals.',
        'fatigue_rest' => 'Listen to your body. Take short breaks and prioritize rest.',
    ],
];
