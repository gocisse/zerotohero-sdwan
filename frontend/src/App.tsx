import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import Layout from './components/Layout'
import CourseProvider from './course/CourseProvider'
import GlossaryProvider from './glossary/GlossaryProvider'
import ProgressProvider from './progress/ProgressProvider'
import GlossaryPage from './pages/GlossaryPage'
import Home from './pages/Home'
import ProgressPage from './pages/ProgressPage'
import QuizPage from './pages/QuizPage'
import ReviewPage from './pages/ReviewPage'
import LessonPage from './pages/LessonPage'
import NotFound from './pages/NotFound'
import SectionPage from './pages/SectionPage'

export default function App() {
  return (
    <BrowserRouter>
      <CourseProvider>
        <GlossaryProvider>
          <ProgressProvider>
            <Routes>
              <Route element={<Layout />}>
                <Route index element={<Home />} />
                <Route path="course" element={<Navigate to="/" replace />} />
                <Route path="course/:sectionId" element={<SectionPage />} />
                <Route path="course/:sectionId/quiz" element={<QuizPage />} />
                <Route path="course/:sectionId/:lessonId" element={<LessonPage />} />
                <Route path="progress" element={<ProgressPage />} />
                <Route path="glossary" element={<GlossaryPage />} />
                <Route path="review" element={<ReviewPage />} />
                <Route path="*" element={<NotFound />} />
              </Route>
            </Routes>
          </ProgressProvider>
        </GlossaryProvider>
      </CourseProvider>
    </BrowserRouter>
  )
}
