@if (session('status'))
    <div class="alert success">{{ session('status') }}</div>
@endif

@if (session('error'))
    <div class="alert error">{{ session('error') }}</div>
@endif

@if ($errors->any() && ! $errors->has('__form'))
    <div class="alert error">
        @foreach ($errors->all() as $error)
            <div>{{ $error }}</div>
        @endforeach
    </div>
@endif
