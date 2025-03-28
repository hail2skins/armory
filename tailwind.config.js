/** @type {import('tailwindcss').Config} */
module.exports = {
    content: [
               "./cmd/web/**/*.html", "./cmd/web/**/*.templ",
    ],
    theme: {
        extend: {
            colors: {
                gunmetal: {
                    100: '#F3F4F6',
                    200: '#E5E7EB',
                    300: '#D1D5DB',
                    400: '#9CA3AF',
                    500: '#6B7280',
                    600: '#4B5563',
                    700: '#374151',
                    800: '#1F2937',
                    900: '#111827',
                },
                brass: {
                    100: '#FEF3C7',
                    200: '#FDE68A',
                    300: '#FCD34D',
                    400: '#FBBF24',
                    500: '#F59E0B',
                    600: '#D97706',
                    700: '#B45309',
                    800: '#92400E',
                    900: '#78350F'
                }
            }
        },
    },
    plugins: [],
}

