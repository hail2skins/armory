/** @type {import('tailwindcss').Config} */
module.exports = {
    content: [
               "./cmd/web/**/*.html", "./cmd/web/**/*.templ",
    ],
    theme: {
        extend: {
            colors: {
                gunmetal: {
                    700: '#374151',
                    800: '#1F2937',
                    900: '#111827',
                },
                brass: {
                    300: '#FCD34D',
                    400: '#FBBF24',
                }
            }
        },
    },
    plugins: [],
}

