package test

import (
	"testing"
	"time"

	"github.com/jaracil/ei"
	nexus "github.com/nayarsystems/nxgo/nxcore"
)

func TestTopicBadPipe(t *testing.T) {
	conn, err := login(UserA, UserA)
	if err != nil {
		t.Fatalf("sys.login userA: %s", err.Error())
	}
	defer conn.Close()
	_, err = conn.TopicSubscribe(&nexus.Pipe{}, Prefix4)
	if err == nil {
		t.Errorf("topic.sub bad pipe: expecting error")
	}
	_, err = conn.TopicUnsubscribe(&nexus.Pipe{}, Prefix4)
	if err == nil {
		t.Errorf("topic.unsub bad pipe: expecting error")
	}
}

func TestTopicNobodySubscribed(t *testing.T) {
	conn, err := login(UserA, UserA)
	if err != nil {
		t.Fatalf("sys.login userA: %s", err.Error())
	}
	defer conn.Close()
	res, err := conn.TopicPublish(Prefix4, "my hello")
	if err != nil {
		t.Errorf("topic.publish: %s", err.Error())
	}
	if ei.N(res).M("sent").IntZ() != 0 {
		t.Errorf("topic.publish: expecting nobody listening")
	}
}

func TestTopicSubscribePublish(t *testing.T) {
	sub1conn, err := login(UserA, UserA)
	if err != nil {
		t.Fatalf("sys.login userA: %s", err.Error())
	}
	sub2conn, err := login(UserB, UserB)
	if err != nil {
		t.Fatalf("sys.login userB: %s", err.Error())
	}
	pub1conn, err := login(UserC, UserC)
	if err != nil {
		t.Fatalf("sys.login userC: %s", err.Error())
	}
	pub2conn, err := login(UserD, UserD)
	if err != nil {
		t.Fatalf("sys.login userD: %s", err.Error())
	}

	// Subscribe
	rpipe1, err := sub1conn.PipeCreate()
	if err != nil {
		t.Errorf("pipe.create: %s", err.Error())
	}
	_, err = sub1conn.TopicSubscribe(rpipe1, Prefix4)
	if err != nil {
		t.Errorf("topic.sub: %s", err.Error())
	}
	rpipe2, err := sub2conn.PipeCreate()
	if err != nil {
		t.Errorf("pipe.create: %s", err.Error())
	}
	_, err = sub2conn.TopicSubscribe(rpipe2, Prefix4)
	if err != nil {
		t.Errorf("topic.sub: %s", err.Error())
	}

	// Publish
	if _, err = pub1conn.TopicPublish(Prefix4, 1); err != nil {
		t.Errorf("topic.pub: %s", err.Error())
	}
	if _, err = pub1conn.TopicPublish(Prefix4, 2); err != nil {
		t.Errorf("topic.pub: %s", err.Error())
	}
	if _, err = pub1conn.TopicPublish(Prefix4, 3); err != nil {
		t.Errorf("topic.pub: %s", err.Error())
	}
	if _, err = pub1conn.TopicPublish(Prefix4, 4); err != nil {
		t.Errorf("topic.pub: %s", err.Error())
	}

	time.Sleep(time.Millisecond * 100)

	// Read
	topicData, err := rpipe1.TopicRead(10, time.Second*5)
	if err != nil {
		t.Errorf("pipe.read from topic: %s", err.Error())
	}
	if len(topicData.Msgs) != 4 {
		t.Errorf("pipe.read from topic: expecting 4 messages: got %d", len(topicData.Msgs))
	} else {
		for i := 0; i < 4; i++ {
			mn := ei.N(topicData.Msgs[i].Msg).IntZ()
			if mn != i+1 {
				t.Errorf("pipe.read from topic: expecting message %d got %d", i+1, mn)
			}
		}
	}

	// Un/subscribe other pipes
	_, err = sub1conn.TopicUnsubscribe(rpipe2, Prefix4)
	if err != nil {
		t.Errorf("topic.unsub with other pipe: %s", err.Error())
	}
	_, err = sub1conn.TopicSubscribe(rpipe2, Prefix4)
	if err != nil {
		t.Errorf("topic.sub other pipe: %s", err.Error())
	}

	time.Sleep(time.Millisecond * 200)

	// Unsubscribe and subscribe again
	pub2conn.TopicPublish(Prefix4, 1000)
	_, err = sub1conn.TopicUnsubscribe(rpipe1, Prefix4)
	if err != nil {
		t.Errorf("topic.unsub: %s", err.Error())
	}
	time.Sleep(time.Millisecond * 200)
	pub2conn.TopicPublish(Prefix4, 2000)
	if _, err = sub1conn.TopicSubscribe(rpipe1, Prefix4); err != nil {
		t.Errorf("topic.sub: %s", err.Error())
	}
	time.Sleep(time.Millisecond * 200)
	pub2conn.TopicPublish(Prefix4, 4000)
	time.Sleep(time.Millisecond * 200)
	topicData, err = rpipe1.TopicRead(10, time.Second*5)
	if err != nil {
		t.Errorf("pipe.read from topic: %s", err.Error())
	}
	if len(topicData.Msgs) != 2 {
		t.Errorf("pipe.read from topic: expecting 2 messages: got %d", len(topicData.Msgs))
	}
	if msg1 := ei.N(topicData.Msgs[0].Msg).IntZ(); msg1 != 1000 {
		t.Errorf("pipe.read from topic: expecting message 1000 got %d", msg1)
	}
	if msg2 := ei.N(topicData.Msgs[1].Msg).IntZ(); msg2 != 4000 {
		t.Errorf("pipe.read from topic: expecting message 4000 got %d", msg2)
	}

	// Unsubscribe and read
	if _, err = sub2conn.TopicUnsubscribe(rpipe2, Prefix4); err != nil {
		t.Errorf("topic.unsub: %s", err.Error())
	}
	time.Sleep(time.Millisecond * 100)
	topicData, err = rpipe2.TopicRead(10, time.Second*5)
	if len(topicData.Msgs) != 7 {
		t.Errorf("pipe.read from topic: expecting 7 messages got %d", len(topicData.Msgs))
	}

	// Close pipe and read
	pub1conn.TopicPublish(Prefix4, 8000)
	if _, err = rpipe1.Close(); err != nil {
		t.Errorf("pipe.close: %s", err.Error())
	}
	pub1conn.TopicPublish(Prefix4, 16000)
	topicData, err = rpipe1.TopicRead(10, time.Second*5)
	if err == nil {
		t.Errorf("pipe.read from topic on closed pipe: expecting error")
	}

	time.Sleep(time.Second * 1)
	pub1conn.Close()
	pub2conn.Close()
	sub1conn.Close()
	sub2conn.Close()
}
