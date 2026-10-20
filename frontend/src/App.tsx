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
import ChallengePage from './sim/ChallengePage'
import EdgePage from './sim/EdgePage'
import EdgesPage from './sim/EdgesPage'
import MapPage from './sim/MapPage'
import MonitorPage from './sim/MonitorPage'
import OverlayChallengePage from './sim/OverlayChallengePage'
import PathExplainerPage from './sim/PathExplainerPage'
import ReachabilityPage from './sim/ReachabilityPage'
import RoutesPage from './sim/RoutesPage'
import RoutingChallengePage from './sim/RoutingChallengePage'
import SimLayout from './sim/SimLayout'
import TimelinePage from './sim/TimelinePage'
import TopologyLayout from './sim/TopologyLayout'
import TopologyPicker from './sim/TopologyPicker'
import WhatIfPage from './sim/WhatIfPage'

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
                <Route path="sim" element={<SimLayout />}>
                  <Route index element={<TopologyPicker />} />
                  <Route path=":topologyId" element={<TopologyLayout />}>
                    <Route index element={<EdgesPage />} />
                    <Route path="edges/:edgeId" element={<EdgePage />} />
                    <Route path="what-if" element={<WhatIfPage />} />
                    <Route path="challenge" element={<ChallengePage />} />
                    <Route path="map" element={<MapPage />} />
                    <Route path="monitor" element={<MonitorPage />} />
                    <Route path="paths" element={<PathExplainerPage />} />
                    <Route path="timeline" element={<TimelinePage />} />
                    <Route path="overlay-challenge" element={<OverlayChallengePage />} />
                    <Route path="routes" element={<RoutesPage />} />
                    <Route path="reachability" element={<ReachabilityPage />} />
                    <Route path="routing-challenge" element={<RoutingChallengePage />} />
                  </Route>
                </Route>
                <Route path="*" element={<NotFound />} />
              </Route>
            </Routes>
          </ProgressProvider>
        </GlossaryProvider>
      </CourseProvider>
    </BrowserRouter>
  )
}
